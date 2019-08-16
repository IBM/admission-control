package webhook

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
	//	"k8s.io/client-go/rest"
)

// getAdmissionWhConfig reads webhook config data from configmap
func getAdmissionWhConfig(pathname string) (AdmissionWhConfig, error) {

	var admissionConf AdmissionWhConfig
	dat, err := ioutil.ReadFile(pathname)
	if err != nil {
		logt.Error(err, "Webhook config read file error", pathname)
		return admissionConf, err
	}

	if err := json.Unmarshal(dat, &admissionConf); err != nil {
		logt.Error(err, "Immutables config unmarshal error")
		return admissionConf, err
	}
	logt.Info("webhook data", "config", fmt.Sprintf("%v", admissionConf))
	return admissionConf, nil

}

// getImmutablesConfig reads immutables from configmap
func getImmutablesConfig(pathname string) ([]ImmutablesConfig, error) {

	var immutablesConf []ImmutablesConfig
	dat, err := ioutil.ReadFile(pathname)
	if err != nil {
		logt.Error(err, "Immutable config read file error")
		return immutablesConf, err
	}

	if err := json.Unmarshal(dat, &immutablesConf); err != nil {
		logt.Error(err, "Immutables config unmarshal error")
		return immutablesConf, err
	}
	return immutablesConf, nil

}

// getLabelsConfig reads required labels  from configmap
func getLabelsConfig(pathname string) ([]LabelsConfig, error) {

	var labelsConf []LabelsConfig
	logt.Info("configmap labels", "filepath", pathname)
	dat, err := ioutil.ReadFile(pathname)
	if err != nil {
		logt.Error(err, "Labels config read file error")
		return labelsConf, err
	}

	if err := json.Unmarshal(dat, &labelsConf); err != nil {
		logt.Error(err, "Labels configmap unmarshal erro")
		return labelsConf, err
	}
	logt.Info("Labels config data", "labels", fmt.Sprintf("%v", labelsConf))
	return labelsConf, nil

}

// validateLabels validates if the availableLabels contains the required labels for the resource kind
func validateLabels(kind string, availableLabels map[string]string) (bool, string) {
	allowed := true
	var message string
	var requiredLabels []string

	// get labels config
	labelsConfig, err := getLabelsConfig(LabelsConfigPath)
	if err != nil {
		return allowed, "no required labels are found"
	}

	for _, lconfig := range labelsConfig {
		logt.Info("look up labels requirement", "config_kind", lconfig.Kind, "request_kind", kind)
		if kind == lconfig.Kind {
			requiredLabels = lconfig.Labels
			break
		}
	}
	logt.Info("check required labels", "required", fmt.Sprintf("%v", requiredLabels), "available", fmt.Sprintf("%v", availableLabels))
	for _, rl := range requiredLabels {
		if _, ok := availableLabels[rl]; !ok {
			allowed = false
			message = "required labels are not set: " + rl
			break
		}
	}
	return allowed, message
}

// Parse parses a json map and returns a flat map[string] with all keys in lowercase
func Parse(key string, value interface{}) (map[string]interface{}, error) {
	newjson := make(map[string]interface{})

	if jsonobj, ok := value.(map[string]interface{}); ok {
		//fmt.Printf("%s is a map[string]: %v\n", key, jsonobj)
		for key1, value1 := range jsonobj {
			//	fmt.Printf("  key1=%v value1=%v\n", key1, value1)
			newkey := key + "." + key1
			newjson[strings.ToLower(newkey)] = value1
			if _, ok2 := value1.(map[string]interface{}); ok2 {
				newjson2, _ := Parse(newkey, value1)
				for k, v := range newjson2 {
					newjson[strings.ToLower(k)] = v
				}
			}
		}
		return newjson, nil
	}
	return nil, nil
}

// isUpdateable calls ibmcloud catalog to check if a service is plan updateable
func isUpdateable(servicename string) bool {
	encoded := &url.URL{Path: servicename}
	//logt.Info("in isUpdateable()", "encoded_servicename", encoded.String())
	uri := IBMCloudCatalogURI + "?q=" + encoded.String()
	logt.Info("calling ibmcloud catalog", "uri", uri)
	resp, err := restCallFunc(uri, nil, "GET", "", "", true)
	if err != nil || resp.StatusCode != 200 {
		logt.Error(err, "call to ibmcloud catalog failed", "servicename", servicename)
		return false
	}
	//fmt.Printf("resp:  %v", resp)
	mybyte := []byte(resp.Body)
	mycatalog := CloudCatalog{}
	json.Unmarshal(mybyte, &mycatalog)
	upgradeableServices := getPlanUpdateables(mycatalog.Resources)
	if len(upgradeableServices) > 0 {
		return true
	}
	return false
}

// getPlanUpdateables parses the resources for services whose plan is updateable and returns them as an UpdateableService array
func getPlanUpdateables(resources []ResourceC) []UpdateableService {
	var upgradeableServices = []UpdateableService{}
	for i := range resources {
		if resources[i].Kind == "service" {
			logt.Info("in isUpdateable()", "name", resources[i].Name, "id", resources[0].ID, "display_name", resources[0].Overview.Engish.DisplayName, "plan_updateable", resources[0].Metadata.Service.PlanUpdateable)
			if resources[0].Metadata.Service.PlanUpdateable {
				service := UpdateableService{resources[i].Name, resources[0].Overview.Engish.DisplayName, resources[0].ID}
				upgradeableServices = append(upgradeableServices, service)
			}
		}
	}
	return upgradeableServices
}

// restCallFunc makes a rest call
func restCallFunc(rsString string, postBody []byte, method string, header string, token string, expectReturn bool) (RestResult, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	restClient := http.Client{
		Timeout:   time.Second * 15,
		Transport: tr,
	}
	u, _ := url.ParseRequestURI(rsString)
	urlStr := u.String()
	var req *http.Request
	if postBody != nil {

		req, _ = http.NewRequest(method, urlStr, bytes.NewBuffer(postBody))
	} else {
		req, _ = http.NewRequest(method, urlStr, nil)
	}

	if token != "" {
		if header == "" {
			req.Header.Set("Authorization", token)
		} else {
			req.Header.Set(header, token)
		}
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := restClient.Do(req)
	if err != nil {
		return RestResult{}, err
	}
	defer res.Body.Close()

	if expectReturn {
		body, err := ioutil.ReadAll(res.Body)
		result := RestResult{StatusCode: res.StatusCode, Body: string(body[:])}
		return result, err
	}
	return RestResult{}, nil
}
