# How to Configure Admission Control for Your CRDs


## 1. Download Admission Control YAML Files 

The configuration and deployment of the admission control is specified in a collection of yaml files. Run the following command to create a directory on your local machine, then download these yaml files from github.com. 

```
mkdir admission-control
cd admission-control
curl -sL https://raw.githubusercontent.com/IBM/admission-control/master/hack/downloadrelease.sh | bash 
```
The download yaml files, as listed below, reside in `releases/v0.1.0/` directory. Among them three files, shown in **bold font**, need to be updated for your CRDs. The sections below explain in details.

|YAML File  |      Comment      | 
|-----------|-------------------|
| 000_namespace.yaml | the namespace in which admission control deployment runs |
| 001_serviceaccount.yaml | the service account to run admission control |
| **002_rbac_role.yaml** | the role with access permissions needed for admission control |
| 003_rbac_role_binding.yaml | the binding that grants the role to the service account |
| **004_validation_rules_configmap.yaml** | the configmap with the custom validation rules for CRDs |
| 005_placeholder_secret.yaml | an empty secret for admission control server to store its TLS certificates for communication with k8s API-servcer |
| 006_mutate_wh_configmap.yaml | the configmap with configuration data for mutating webhook |
| **007_validate_wh_configmap.yaml** | the configmap with configuration data for validating webhook |
| 008_service.yaml | the service for accessing admission control deployment |
| 009_statefulset.yaml | the stateful set for admission control deployment |


## 2. Grant Access Permissions to Your CRDs - 002_rbac_role.yaml 

Admission control server requires `get` and `list` access to the resources in order to perform validation. The access permissions are defined as `rules` in a ClusterRole. 

Edit **002_rbac_role.yaml**, add new rules with `get` and `list` permissions to your CRDs, and save the changes. For example, the following listing shows the rule for any resource kinds (CRDs) in `ibmcloud.ibm.com` apiGroup. 
    
```
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  creationTimestamp: null
  name: admission-control-role
  labels:
    app.kubernetes.io/name: admission-control
rules:
- apiGroups: ...
- apiGroups: ...
...
- apiGroups:
  - ibmcloud.ibm.com
  resources:
  - '*'
  verbs:
  - get
  - list
```


 ## 3. Update Webhook Registration - 007_validate_wh_configmap.yaml

Upon deployment the admission control server automatically registers two webhooks with k8s API server, one for validating and the other for mutating. The mutating webhook is not used for now. The discussion below focuses on the validating webhook. 

The admission control reads the validating webhook configuration data from the ConfigMap as specified in **007_validate_wh_configmap.yaml**. The value of `data.config` is a JSON string representation of the configuration data, see the sample listing below. 

```
apiVersion: v1
kind: ConfigMap
metadata:
  name: validate-wh-config
  namespace: admission-control
  labels:
    app.kubernetes.io/name: admission-control
data:
  config: '{"name": "validating-webhook", "rules": [{"operations": ["CREATE", "UPDATE"], "apiGroups": ["ibmcloud.ibm.com"], "apiVersions": ["v1alpha1"], "resources": ["services", "bindings", "esindices", "topics"]}]}'

```

* `name` - the webhook name to be registered on k8s API server

* `rules` - array of rules specifying when the admission control will be called for review and approval. Each rule consists of:

    * `operations` - an array of operations such as CREATE, UPDATE, and/or DELETE
    
    * `apiGroups` -  an array of API groups of your CRDs
    
    * `apiVersion` - an array of API versions of your CRDs 
    
    * `resources` - an array of plural names of your CRDs

Edit **007_validate_wh_configmap.yaml** with the rules for your CRDs, and save the changes.




 
## 4. Specify Custom Validation Rules - 004_validation_rules_configmap.yaml
    
Admission Control reads the validation rules from the ConfigMap specified in **004_validation_rules_configmap.yaml**, see the listing below. It contains rules for various validation types such as immutables, mutaul exclusions, and labels (`data.immutables`, `data.exclusives`, and `data.labels`). The rules for each validation type are stored as value in a JSON string representation. 

```
apiVersion: v1
kind: ConfigMap
metadata:
  name: validation-rules-config
  namespace: admission-control
  labels:
    app.kubernetes.io/name: admission-control
data:
  exclusives: '[{"kind":"EsIndex","group":"ibmcloud.ibm.com","version":"v1alpha1","exclusives":[["Spec.BindingFrom","Spec.esURIComposed"]]}]'
  immutables: '[{"kind":"EsIndex","group":"ibmcloud.ibm.com","version":"v1alpha1","immutables":["Spec.BindingFrom.Name","Spec.IndexName","Spec.NumberOfShards","Spec.NumberOfReplicas"]},{"kind":"Service","group":"ibmcloud.ibm.com","version":"v1alpha1","immutables":["Spec.Plan","Spec.ServiceClass"]},{"kind":"Binding","group":"ibmcloud.ibm.com","version":"v1alpha1","immutables":["Spec.Role","Spec.ServiceNamespace","Spec.ServiceName"]}]'
  labels: '[{"kind":"EsIndex","group":"ibmcloud.ibm.com","version":"v1alpha1","labels":["ibmcloud.ibm.com/instance","ibmcloud.ibm.com/name"]}]'
```

**Rule for Immutables**

* `kind` - CRD kind

* `group` - CRD API group

* `version` - CRD API version

* `immutables` - an array of spec names their values are immutable

**Rule for Mutual Exclusions**

* `kind` - CRD kind

* `group` - CRD API group
  
* `version` - CRD API version

* `exclusives` - an array of arrays of mutually exclusive specs

**Rule for Labels**

* `kind` - CRD kind
 
* `group` - CRD API group

* `version` - CRD API version

* `labels` - an array of required labels

Edit **004_validation_rules_configmap.yaml** with the validation rules for your CRDs, and save the changes. 

In case where you have not have any rules for a validation type, you may have an empty array, see the example below.

```
data:
  exclusives: '[]'
  immutables: '[{"kind":"Service","group":"ibmcloud.ibm.com","version":"v1alpha1","immutables":["Spec.Plan","Spec.ServiceClass"]},{"kind":"Binding","group":"ibmcloud.ibm.com","version":"v1alpha1","immutables":["Spec.Role","Spec.ServiceNamespace","Spec.ServiceName"]}]'
  labels: '[]'
```

## 5. Deploy Your Customized Admission Control

Once you have completed steps 2-4, you may deploy the admission control with the following command (assuming all the yaml files are under directory `releases/v0.1.0`).

```
kubectl apply -f releases/v0.1.0/
```

Run the following commands to verify the deployment.

```
kubectl get pod -n admission-control
kubectl get validatingwebhookconfiguration
```

## 6. Test

Now you may test the admission control, for example, for immutable validation.

1. create a new resource (e.g. xxx.yaml) of your CRD knid

2. update the value of an immutable spec in the yaml file

3. apply the update yaml. 

The update request should receive a rejection from k8s API server. 

The list bellow shows an example rejection response where the request attempted to change the plan of an elasticsearch resource (IBM Cloud Service). The last line shows the reason for rejection. 

```
$ kubectl apply -f elasticsearch.yaml 
Error from server: error when applying patch:
{"metadata":{"annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"ibmcloud.ibm.com/v1alpha1\",\"kind\":\"Service\",\"metadata\":{\"annotations\":{},\"name\":\"es-test\",\"namespace\":\"default\"},\"spec\":{\"context\":{\"org\":\"seed-test\",\"region\":\"us-east\",\"resourcegroup\":\"default\",\"resourcelocation\":\"us-east\",\"space\":\"test\"},\"plan\":\"enterprise\",\"serviceClass\":\"databases-for-elasticsearch\"}}\n"}},"spec":{"plan":"enterprise"}}
to:
Resource: "ibmcloud.ibm.com/v1alpha1, Resource=services", GroupVersionKind: "ibmcloud.ibm.com/v1alpha1, Kind=Service"
Name: "es-test", Namespace: "default"
Object: &{map["apiVersion":"ibmcloud.ibm.com/v1alpha1" "kind":"Service" "metadata":map["creationTimestamp":"2019-08-13T13:56:54Z" "finalizers":["service.ibmcloud.ibm.com"] "generation":'\x01' "namespace":"default" "selfLink":"/apis/ibmcloud.ibm.com/v1alpha1/namespaces/default/services/es-test" "uid":"357645d0-bdd2-11e9-8b83-e6574b36caee" "annotations":map["kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"ibmcloud.ibm.com/v1alpha1\",\"kind\":\"Service\",\"metadata\":{\"annotations\":{},\"name\":\"es-test\",\"namespace\":\"default\"},\"spec\":{\"context\":{\"org\":\"seed-test\",\"region\":\"us-east\",\"resourcegroup\":\"default\",\"resourcelocation\":\"us-east\",\"space\":\"test\"},\"plan\":\"standard\",\"serviceClass\":\"databases-for-elasticsearch\"}}\n"] "name":"es-test" "resourceVersion":"16879940"] "spec":map["context":map["resourcelocation":"us-east" "space":"test" "org":"seed-test" "region":"us-east" "resourcegroup":"default"] "plan":"standard" "serviceClass":"databases-for-elasticsearch"] "status":map["state":"Online" "context":map["org":"seed-test" "region":"us-east" "resourcegroup":"default" "resourcelocation":"us-east" "space":"test"] "instanceId":"crn:v1:bluemix:public:databases-for-elasticsearch:us-east:a/948beb24642359e015b0a93c4000cb6c:886859ef-4e27-481b-8485-561d5f1684d8::" "message":"Online" "plan":"standard" "serviceClass":"databases-for-elasticsearch" "serviceClassType":""]]}
for: "elasticsearch.yaml": admission webhook "custom.admission.control" denied the request: spec.plan is immutable. 
```
 