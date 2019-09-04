# Admission Control for k8s CRDs
The project provides a k8s admission control webhook server to perform custom validations of resource requests. Specifically it provides the following capabilities.

* **Validation of Immutables** Some resource kinds may have contraints in whether their specs can be dynamically updated. For example database related CRDs may limit the change to storage allocation; cloud services may not support dynamic subscription plan upgrade/downgrade due to provisioning constraints. **Validation of immutables** allows you to specify immutability of CRD spec parameters. The admission control enforces these immutable rules by rejecting any requests that violate the rulels. 

* **Validation of Mutual Exclusions** Some resource kinds may have optional parameters in their specifications that are mutually exclusive. For example custom resources like [Event Streams Topics](https://github.com/IBM/event-streams-topic) support mutiple credential source options in specification. These options are mutually exclusive, and only one of them should be present in a yaml request. **Validation of Mutual Exclusion** allows you to specify rules for mutually exclusive CRD spec parameters. The admission control enforces the rules by rejecting any requests that contain mutually exclusive specs with a message for correction. 

* **Validation of Labels** validates the required labels according to user specified labeling rules. The admission control enforces the rules by rejecting any requests that violate the rulles.

## 1. Install

Run the following command to install admission control:

```
curl -sL https://raw.githubusercontent.com/IBM/admission-control/master/hack/install.sh | bash 
```

It will install the latest admission control server on your cluster under namespace `admission-control`. The table below lists all the resources deployed including the default configuration for webhook registrations and validation rules for [IBM Cloud Service CRDs](https://github.com/IBM/cloud-operators). Section 3 below explains how to update the configuration to work with your own CRDs.

|Name  |      Kind      |  Namespace | Comment |
|----------|:-------------:|:------:|-----------|
| admission-control |  Namespace | - |  |
| admission-control |   ServiceAccount  | admission-control |  |
| admission-control-role | ClusterRole | - | Access permissions for admission control |
| admission-control-rolebinding | ClusterRoleBinding | - |  |
| validation-rules-config | ConfigMap | admission-control | Validation rules for immutables, mutual exclusives, and labels |
| admission-webhook-certs | Secret | admission-control | Certs for secure connection between admission control and k8s APIServcer |
| validate-wh-config | ConfigMap | admission-control | Validating webhook config data |
| mutate-wh-config | ConfigMap | admission-control | Mutating webhook config data |
| admission-control-service | Service | admission-control |  |
| admission-control  | StatefulSet | admission-control | Admission control server |
| validating-webhook | ValidatingWebhookConfiguration |  | Validating webhook registered with k8s API server by admission control server |
| mutating-webhook | MutatingWebhookConfiguration |  | Mutating webhook registered with k8s API server by admission control server |

View logs to confirm installation:

```
kubectl logs --follow admission-control-0 -n  admission-control
```
    
The log would show something like this:

```
{"level":"info","ts":1566316905.0751424,"logger":"entrypoint","msg":"setting up client for manager"}
{"level":"info","ts":1566316905.0753875,"logger":"entrypoint","msg":"setting up manager"}
{"level":"info","ts":1566316905.4753027,"logger":"entrypoint","msg":"Registering Components."}
{"level":"info","ts":1566316905.4753418,"logger":"entrypoint","msg":"Starting the webwook."}
{"level":"info","ts":1566316905.4753497,"logger":"admission","msg":"env variable settings","ADMISSION_CONTROL_LABELS":"false"}
{"level":"info","ts":1566316905.4753656,"logger":"admission","msg":"env variable settings","ADMISSION_CONTROL_IMMUTABLES":"true"}
{"level":"info","ts":1566316905.4753711,"logger":"admission","msg":"env variable settings","ADMISSION_CONTROL_MUTATION":"true"}
{"level":"info","ts":1566316905.4755173,"logger":"admission","msg":"buildWebhook","path":"/validate","rules":[{"operations":["CREATE","UPDATE"],"apiGroups":["ibmcloud.ibm.com"],"apiVersions":["v1alpha1"],"resources":["services","bindings","esindices","topics"]}]}
{"level":"info","ts":1566316905.4756217,"logger":"admission","msg":"buildWebhook","path":"/mutate","rules":[{"operations":["CREATE","UPDATE"],"apiGroups":["ibmcloud.ibm.com"],"apiVersions":["*"],"resources":["*"]},{"operations":["CREATE"],"apiGroups":["ibmcloud.ibm.com"],"apiVersions":["v1alpha1"],"resources":["bucket"]}]}
{"level":"info","ts":1566316905.4756362,"logger":"admission","msg":"webhook server settings","name":"custom-admission-webhook","namespace":"admission-control","secret name":"admission-webhook-certs","service name":"admission-control-service","cert dir":"/tmp/cert"}
{"level":"info","ts":1566316905.4757433,"logger":"entrypoint","msg":"Starting the Cmd."}
{"level":"info","ts":1566316905.5762658,"logger":"kubebuilder.webhook","msg":"installing webhook configuration in cluster"}
{"level":"info","ts":1566316906.2954004,"logger":"kubebuilder.webhook","msg":"starting the webhook server."}
```


## 2. Uninstall

Run the following command to uninstall admission control:

```
curl -sL https://raw.githubusercontent.com/IBM/admission-control/master/hack/uninstall.sh | bash 
```

## 3. Configure Admission Control for Your CRDs

There are three steps.

1. Grant the admission control server the permission to access your CRD resources

2. Register with k8s API server for admission approval

3. Create admission rules to be executed by admission control

See the following links for configuration details for each validation features:

* **[Validation for Immutables](https://github.com/IBM/admission-control/blob/master/doc/ConfigImmutables.md)**

* **Validation for Exclusions** (coming soon)
   
* **Validation for Labels** (coming soon)

