package webhook

import (
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IBMCloudCatalogURI is the uri for IBM cloud service catalog. It is used by validator to verify if a service plan is updateable.
const IBMCloudCatalogURI = "https://globalcatalog.cloud.ibm.com/api/v1"

// ValidateWhConfigPath is path where the ConfigMap is mounted for the validating webhook config data
const ValidateWhConfigPath = "/etc/config/validate-wh/config"

// MutateWhConfigPath is path where the ConfigMap is mounted for the validating webhook config data
const MutateWhConfigPath = "/etc/config/mutate-wh/config"

//AdmissionWhConfig - a struct for admission webhook registration data
type AdmissionWhConfig struct {
	Name  string                                            `json:"name"`
	Rules []admissionregistrationv1beta1.RuleWithOperations `json:",inline" protobuf:"bytes,2,opt,name=rule"`
}

// validatingWebhookName - default name for validatingwebhookconfiguration
const validatingWebhookName = "validating-webhook"

// mutatingWebhookName - default name for mutatingwebhookconfiguration
const mutatingWebhookName = "mutating-webhook"

// LabelsConfigPath is path where the ConfigMap is mounted for the required labels
const LabelsConfigPath = "/etc/config/validation/labels"

// ImmutablesConfigPath is path where the ConfigMap is mounted for the immutable specifiction fields
const ImmutablesConfigPath = "/etc/config/validation/immutables"

// ExclusivesConfigPath is path where the ConfigMap is mounted for the mutual exclusive specifiction fields
const ExclusivesConfigPath = "/etc/config/validation/exclusives"

//LabelsConfig - a struct for required labels per resource type
type LabelsConfig struct {
	Kind   string   `json:"kind"`
	Labels []string `json:"labels,omitempty"`
}

//ImmutablesConfig - a struct for immutable rules
type ImmutablesConfig struct {
	Kind       string   `json:"kind"`
	Immutables []string `json:"immutables,omitempty"`
}

//ExclusivesConfig - a struct for mutual exclusion rules
type ExclusivesConfig struct {
	Kind       string     `json:"kind"`
	Group      string     `json:"group"`
	Version    string     `json:"version,omitempty"`
	Exclusives [][]string `json:"exclusives,omitempty"`
}

//Dependent identifies a dependent resource in cluster
type Dependent struct {
	Kind string
	Name string
}

//AdmissionObj a general struct for kube request object
type AdmissionObj struct {
	TypeMeta   metav1.TypeMeta   `json:",inline"`
	ObjectMeta metav1.ObjectMeta `json:"metadata"`
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

// ServiceC represents the service as defined in ibmcloud catalog api
type ServiceC struct {
	PlanUpdateable bool `json:"plan_updateable"`
}

//MetadataC represents the resource metadata as defined in ibmcloud catalog api
type MetadataC struct {
	Service  ServiceC `json:"service"`
	Original string   `json:"original_name"`
}

//EnglishC represents the resource overview_ui.en as defined in ibmcloud catalog api
type EnglishC struct {
	DisplayName string `json:"display_name"`
}

//OverviewC represents the resource overview_ui as defined in ibmcloud catalog api
type OverviewC struct {
	Engish EnglishC `json:"en"`
}

//ResourceC represents the resource as defined in ibmcloud catalog api
type ResourceC struct {
	Metadata MetadataC `json:"metadata"`
	Overview OverviewC `json:"overview_ui"`
	Kind     string    `json:"kind"`
	ID       string    `json:"id"`
	Name     string    `json:"name"`
}

//CloudCatalog represents the calalog listing as defined in ibmcloud catalog api
type CloudCatalog struct {
	Count     float64     `json:"count"`
	Next      string      `json:"next"`
	Resources []ResourceC `json:"resources"`
}

//UpdateableService represents a cloud service that can be upgraded dynamically
type UpdateableService struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	ID          string `json:"id"`
}

// RestResult is a struct for REST call result
type RestResult struct {
	StatusCode int
	Body       string
	ErrorType  string
}
