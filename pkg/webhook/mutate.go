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
	"net/http"
	"strings"

	appsv1 "k8s.io/api/apps/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

// AdmissionMutator - Mutating webhook for seed resources
type AdmissionMutator struct {
	client client.Client
}

// AdmissionMutator implements admission.Handler.
var _ admission.Handler = &AdmissionMutator{}

func mutate(ctx context.Context, deploy *appsv1.Deployment) error {
	//log.Printf("-------  enter mutate(), deploy name=%v label=%v\n", deploy.ObjectMeta.Name, deploy.ObjectMeta.Namespace)
	//deploy.ObjectMeta.Name = deploy.ObjectMeta.Name + "-seed"
	if deploy.ObjectMeta.GetAnnotations() == nil {
		deploy.ObjectMeta.Annotations = map[string]string{}
	}
	deploy.ObjectMeta.Annotations["seed-admission-control-mutation"] = "testing"
	return nil
}

// Handle - handle mutating admission control requests
func (m *AdmissionMutator) Handle(ctx context.Context, req types.Request) types.Response {
	logt.Info("mutate is called")
	switch strings.ToLower(req.AdmissionRequest.Kind.Kind) {
	case "deployment":
		deploy := &appsv1.Deployment{}
		json.Unmarshal(req.AdmissionRequest.Object.Raw, &deploy)

		// Do deepcopy before actually mutate the object.
		copy := deploy.DeepCopy()
		err := mutate(ctx, copy)
		if err != nil {
			return admission.ErrorResponse(http.StatusInternalServerError, err)
		}
		return admission.PatchResponse(deploy, copy)
	default:
		obj := &appsv1.Deployment{}
		obj2 := &appsv1.Deployment{}
		return admission.PatchResponse(obj, obj2)
	}
}

// AdmissionMutator implements inject.Client.
var _ inject.Client = &AdmissionMutator{}

// InjectClient injects the client into the AdmissionMutator
func (m *AdmissionMutator) InjectClient(c client.Client) error {
	m.client = c
	return nil
}
