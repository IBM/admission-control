# Configure For Immutable Validation

## 1. Grant Permissions to Access Resources 

Admission control server requires `get` and `list` access to the resources in order to perform immutable validation. The access permissions are defined as `rules` in ClusterRole `admission-control-role`. For example, the following listing shows the rule that grants `get` and `list` access to all resource kinds (CRDs) of `ibmcloud.ibm.com` apiGroup. 
    
```
    - apiGroups:
      - ibmcloud.ibm.com
      resources:
      - '*'
      verbs:
      - get
      - list
```

Verify that `admission-control-role` has the rules for your target resource kinds. To add new rules, either run `kubectl edit` command:

```
  $ kubectl edit clusterrole admission-control-role
```
  
or download and edit [002_rbac_role.yaml](https://github.com/IBM/admission-control/blob/master/releases/v0.1.0/002_rbac_role.yaml), then apply the change.

```
  $ kubectl apply -f 002_rbac_role.yaml
```

 ## 2. Update Webhook Registration

Upon deployment the admission control server automatically registers two webhooks with k8s API server, one for validating and the other for mutating. The mutating webhook is not used by the immutable validation. The discussion below focuses on the validating webhook. 

Use the following command to view the validating webhook registration (named `validating-webhook`). Under the element of `webhooks.rules`, it specifies `apiGroups`, `apiVersions`, `operations`, and `resources` for which k8s API server will send the requests to the admission control server for approval. 

```
$ kubectl get validatingwebhookconfiguration validating-webhook -oyaml
apiVersion: admissionregistration.k8s.io/v1beta1
kind: ValidatingWebhookConfiguration
metadata:
  creationTimestamp: 2019-08-21T19:07:29Z
  generation: 1
  name: validating-webhook
  resourceVersion: "28380544"
  selfLink: /apis/admissionregistration.k8s.io/v1beta1/validatingwebhookconfigurations/validating-webhook
  uid: ebd7b6cd-c446-11e9-b3a6-be3413cae007
webhooks:
- admissionReviewVersions:
  - v1beta1
  clientConfig:
    caBundle: LS0tLS1CRUdJTiBDRVJUSU ...... i0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
    service:
      name: admission-control-service
      namespace: admission-control
      path: /validate
  failurePolicy: Ignore
  name: custom.admission.control
  namespaceSelector:
    matchExpressions:
    - key: control-plane
      operator: DoesNotExist
  rules:
  - apiGroups:
    - ibmcloud.ibm.com
    apiVersions:
    - v1alpha1
    operations:
    - CREATE
    - UPDATE
    resources:
    - services
    - bindings
    - esindices
    - topics
    scope: '*'
  sideEffects: Unknown
  timeoutSeconds: 30
  ```

Use `kubectl edit` command to update the `webhooks.rules` by adding a new rule (`apiGroups`) or edit the existing one. Once the update is saved, k8s API server will pickup the change automatically.

```
  $ kubectl edit validatingwebhookconfiguration validating-webhook
```
 
## 3. Specify Immutable Rules  
    
Admission Control reads the immutable rules from a ConfigMap named `immutables-config`. The rules are stored in JSON format, see [005_immutable_configmap.yaml](https://github.com/IBM/admission-control/blob/master/releases/v0.1.0/007_immutable_configmap.yaml) for example. A rule contains four elements: `kind`, `group` (APIGroup), `version` (APIVersion), and `immutables`. The `immutables` is an array of spec parameter pathnames. The listing below shows example immutable rules for [IBM Cloud Service CRD](https://github.com/IBM/cloud-operators) and [IBM Cloud EsIndex CRD](https://github.com/IBM/esindex-operator).

![](https://github.com/IBM/admission-control/blob/master/doc/images/immutable-rules.png)

 
Download [005_immutable_configmap.yaml](https://github.com/IBM/admission-control/blob/master/releases/v0.1.0/007_immutable_configmap.yaml), and edit the value of `data` with your immutable rules. Run the following command to apply the change. The admission control will pick up the change automatically, no restart is needed.

```
   $ kubectl apply -f 007_immutable_configmap.yaml
```

## 4. Test

After the three steps above, you may test it by 1) creating a new resource (e.g. xxx.yaml) of your target knid, 2) update the value of an immutable spec in the yaml file, and 3) apply the update. The update request should receive a rejection from k8s API server. 

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
 