# Configure For Immutables

## 1. Grant ClusterRole Access to Resources 

Admission control requires `get` and `list` access permission to the resources in order to perform immutable validation. The ClusterRole `admission-control-role` specifies the permissions. Verify that permissions for the CRD kinds of your resources. For example, the following listing shows the permission for all CRDs in `ibmcloud.ibm.com` APIGroup.
    
    ```
    - apiGroups:
      - ibmcloud.ibm.com
      resources:
      - '*'
      verbs:
      - get
      - list
    ```
  
  Run following command to add new permissions:
  
  ```
  kubectl edit clusterrole admission-control-role
  ```
  
  or download and edit [002_rbac_role.yaml](https://github.com/IBM/admission-control/blob/master/releases/v0.1.0/002_rbac_role.yaml), then
  
  ```
  kubectl apply -f 002_rbac_role.yaml
  ```
 
 ## 2 Specify Immutable Rules  
    Admission Control reads the immutable rules from a ConfigMap named `immutables-config`. The rules are in JSON format, see [007_immutable_configmap.yaml](https://github.com/IBM/admission-control/blob/master/releases/v0.1.0/007_immutable_configmap.yaml). A rule contains four elements: `kind`, `group` (APIGroup), `version` (APIVersion), and `immutables`. The `immutables` is an array of spec parameter pathnames. The listing below shows the example for [IBM Cloud Service CRD](https://github.com/IBM/cloud-operators) and [IBM Cloud EsIndex CRD](https://github.com/IBM/esindex-operator).
 
    ![](https://github.com/IBM/admission-control/blob/master/doc/images/immutable-rules.png)
 
    Download and edit [007_immutable_configmap.yaml](https://github.com/IBM/admission-control/blob/master/releases/v0.1.0/007_immutable_configmap.yaml) with your immutable rules, then run the following command to apply the change.
    
    ```
    kubectl apply -f 007_immutable_configmap.yaml
    ```
  
 