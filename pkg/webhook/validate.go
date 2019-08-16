/*
Copyright 2019 The Seed team.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

// AdmissionValidator - Validating webhook for seed resources
type AdmissionValidator struct {
	client client.Client
}

// AdmissionValidator implements admission.Handler
var _ admission.Handler = &AdmissionValidator{}

// Handle handles admission validation
func (v *AdmissionValidator) Handle(ctx context.Context, req types.Request) types.Response {
	var (
		availableLabels                 map[string]string
		resourceNamespace, resourceName string
		admissionobj                    AdmissionObj
	)
	admissionobj, err := getObjectMeta(req.AdmissionRequest.Object.Raw)
	if err != nil {
		logt.Error(err, "error from unmarshal raw admission request object")
		return types.Response{
			Response: &admissionv1beta1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Message: err.Error(),
				},
			},
		}
	}

	resourceName, resourceNamespace = admissionobj.ObjectMeta.Name, admissionobj.ObjectMeta.Namespace
	availableLabels = admissionobj.ObjectMeta.Labels
	logt.Info("AdmissionReview", "operation", req.AdmissionRequest.Operation, "name", resourceName,
		"namespace", resourceNamespace, "kind", req.AdmissionRequest.Kind.Kind, "group", req.AdmissionRequest.Kind.Group,
		"version", req.AdmissionRequest.Kind.Version, "labels", availableLabels)

	//validate required labels
	if os.Getenv("ADMISSION_CONTROL_LABELS") == "true" {
		//logt.Info("check labels")
		allowed, message := validateLabels(req.AdmissionRequest.Kind.Kind, availableLabels)
		result := &metav1.Status{
			Message: message,
		}
		if allowed == false {
			return types.Response{
				Response: &admissionv1beta1.AdmissionResponse{
					Allowed: allowed,
					Result:  result,
				},
			}
		}
	}

	//validate immutables
	if req.AdmissionRequest.Operation == "UPDATE" && os.Getenv("ADMISSION_CONTROL_IMMUTABLES") == "true" {
		//parse request sepc and convert to a flat map
		//logt.Info("check immutables", "operation", req.AdmissionRequest.Operation)
		allowed, message := v.validateImmutables(req.AdmissionRequest.Kind.Kind, req.AdmissionRequest.Object.Raw)
		result := &metav1.Status{
			Message: message,
		}
		if allowed == false {
			return types.Response{
				Response: &admissionv1beta1.AdmissionResponse{
					Allowed: allowed,
					Result:  result,
				},
			}
		}
	}

	logt.Info("! AdmissionReview is approved")
	return types.Response{
		Response: &admissionv1beta1.AdmissionResponse{
			Allowed: true,
			Result: &metav1.Status{
				Message: "",
			},
		},
	}
}

// AdmissionValidator implements inject.Client.
var _ inject.Client = &AdmissionValidator{}

// InjectClient injects the client into the AdmissionValidator
func (v *AdmissionValidator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// getObjectMeta gets OjbectMeta from raw resource object
func getObjectMeta(data []byte) (AdmissionObj, error) {
	var obj AdmissionObj
	if err := json.Unmarshal(data, &obj); err != nil {
		return obj, err
	}
	return obj, nil
}

// getObjectFromCluster gets resource object from cluster
func (v *AdmissionValidator) getObjectFromCluster(group string, version string, kind string, namespace string, name string) (*unstructured.Unstructured, error) {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   group,
		Kind:    kind,
		Version: version,
	})

	err := v.client.Get(context.Background(), client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, u)
	if err != nil {
		logt.Error(err, "failed to get object from cluster", "name", name, "namespace", namespace, "kind", kind)
		return u, err
	}
	//logt.Info("resource object", "object", fmt.Sprintf("%v", u))
	return u, nil
}

// checkExistence verifys if all dependencies exist
func (v *AdmissionValidator) checkExistence(namespace string, dependencies []Dependent) []Dependent {
	logt.Info("check dependencies existence", "dependencies", fmt.Sprintf("%v", dependencies))
	var notfound []Dependent
	for _, entry := range dependencies {
		_, err := v.getObjectFromCluster("", "v1", entry.Kind, namespace, entry.Name)
		if err != nil {
			notfound = append(notfound, entry)
		}
	}
	logt.Info("dependencies that are not found", "notFound", fmt.Sprintf("%v", notfound))
	return notfound
}

// validateImmutables validates if request has changes to immutable attributes in spec
func (v *AdmissionValidator) validateImmutables(kind string, admobj []byte) (bool, string) {
	allowed := true
	var message string
	var immutables []string

	//parse request sepc and convert to a flat map
	var req map[string]interface{}
	json.Unmarshal(admobj, &req)
	str := strings.Split(req["apiVersion"].(string), "/")
	group, version := str[0], str[1]
	metadata := req["metadata"].(map[string]interface{})
	namespace := metadata["namespace"].(string)
	name := metadata["name"].(string)
	spec, _ := Parse("spec", req["spec"].(map[string]interface{}))
	//logt.Info("flatterned spec json map", "spec", fmt.Sprintf("%v", spec))

	// get immutables from configmap
	immutablesConfig, err := getImmutablesConfig(ImmutablesConfigPath)
	if err != nil {
		return true, "no immutables found"
	}

	for _, imconfig := range immutablesConfig {
		if kind == imconfig.Kind {
			immutables = imconfig.Immutables
			logt.Info("Immutables", "kind", kind, "immutables", fmt.Sprintf("%v", immutables))

			// get existing resource spec
			existingobj, err := v.getObjectFromCluster(group, version, kind, namespace, name)
			if err != nil {
				logt.Error(err, "getObjectFromCluster error")
				return false, "failed to retrieve the existing resource"
			}
			existingspec, _ := Parse("spec", existingobj.Object["spec"].(map[string]interface{}))
			//logt.Info("flatterned existingspec json map", "spec", fmt.Sprintf("%v", existingspec))

			// compare the new spec with the existing for immutables
			for _, item := range immutables {
				logt.Info("immutable", "item", item)
				itemlower := strings.ToLower(item)
				//if !reflect.DeepEqual(spec[itemlower], existingspec[itemlower]) { //!!!this would go panic if the item does not exist.
				if spec[itemlower] != existingspec[itemlower] {
					//logt.Info("existing", "itemlower", fmt.Sprintf("%v", existingspec[itemlower]), "kind", reflect.TypeOf(existingspec[itemlower]).Kind())
					//logt.Info("new     ", "itemlower", fmt.Sprintf("%v", spec[itemlower]), "kind", reflect.TypeOf(spec[itemlower]).Kind())
					/* It appears a bug on kube-apiserver side that formats a int64 data kind (as defined in crd yaml) to float in admission review request.
					This makes the condition above to be true mistakenly when the existing and the new have the same value.
					As a workaround, convert them to string and then compare */
					oldStr := fmt.Sprintf("%v", existingspec[itemlower])
					newStr := fmt.Sprintf("%v", spec[itemlower])
					if oldStr != newStr {
						if itemlower == "spec.plan" && strings.ToLower(kind) == "service" {
							servicename := fmt.Sprintf("%v", existingspec["spec.serviceclass"])
							updateable := isUpdateable(servicename)
							if updateable == false {
								logt.Info("reject update", "spec", item, "existing", existingspec[itemlower], "new", spec[itemlower])
								message += itemlower + " is immutable. "
								allowed = false
							}
						} else {
							logt.Info("reject update", "spec", item, "existing", existingspec[itemlower], "new", spec[itemlower])
							message += itemlower + " is immutable. "
							allowed = false
						}
					}
				}
			}
		}
	}
	if !allowed {
		logt.Info("reject the request", "message", message)
	}
	return allowed, message
}
