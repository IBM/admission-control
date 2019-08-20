#!/bin/bash
#
# Copyright 2019 IBM Corp. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

# Delete the deployment for admission-control 
kubectl delete statefulset -l app.kubernetes.io/name=admission-control -n admission-control

# Delete the serviceaccount for admission-control 
kubectl delete serviceaccount,service -l app.kubernetes.io/name=admission-control -n admission-control

# Delete configmaps and secret for admission-control 
kubectl delete configmap -l app.kubernetes.io/name=admission-control -n admission-control
kubectl delete secret admission-webhook-certs -n admission-control

# Delete the role and role binding for admission-control 
kubectl delete clusterrole,clusterrolebinding -l app.kubernetes.io/name=admission-control  

# Delete admission control webhook configurations
kubectl delete validatingwebhookconfiguration validating-webhook
kubectl delete mutatingwebhookconfiguration mutating-webhook

# Delete namespace
#kubectl delete ns admission-control
