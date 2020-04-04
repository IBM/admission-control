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
	"fmt"
	"os"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	k8types "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/builder"
	admissiontypes "sigs.k8s.io/controller-runtime/pkg/webhook/types"
)

var logt = logf.Log.WithName("admission")

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager) error

// AddToManager adds all Controllers to the Manager
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfigurations;validatingwebhookconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources="",verbs=get;list
// +kubebuilder:rbac:groups="ibmcloud.ibm.com",resources="*",verbs=get;list
func AddToManager(m manager.Manager) error {
	for _, f := range AddToManagerFuncs {
		if err := f(m); err != nil {
			logt.Error(err, "webhook.AddToManagerFuncs error")
			return err
		}
	}

	logt.Info("env variable settings", "ADMISSION_CONTROL_LABELS", os.Getenv("ADMISSION_CONTROL_LABELS"))
	logt.Info("env variable settings", "ADMISSION_CONTROL_IMMUTABLES", os.Getenv("ADMISSION_CONTROL_IMMUTABLES"))
	logt.Info("env variable settings", "ADMISSION_CONTROL_MUTATION", os.Getenv("ADMISSION_CONTROL_MUTATION"))

	err := startWebhookServer(m)
	if err != nil {
		return err
	}
	return nil
}

// startWebhookServer starts admission control webhook server
func startWebhookServer(mgr manager.Manager) error {
	logt.Info("start admission webhook server")

	var rules []admissionregistrationv1beta1.RuleWithOperations
	var whV, whM *admission.Webhook
	var whVName, whMName string

	validateConfig, err := getAdmissionWhConfig(ValidateWhConfigPath)
	if err != nil {
		return fmt.Errorf("failed when read webhook configuration configmap at %v. error: ", ValidateWhConfigPath)
	}
	whVName = validateConfig.Name
	rules = checkRules(validateConfig.Rules)
	if rules != nil {
		whV, err = buildWebhook(mgr, "custom.admission.control", "/validate", admissiontypes.WebhookTypeValidating, rules)
	}

	if os.Getenv("ADMISSION_CONTROL_MUTATION") == "true" {
		mutateConfig, err := getAdmissionWhConfig(MutateWhConfigPath)
		if err != nil {
			logt.Info("no admission webhook configuration configmap is found at %v", MutateWhConfigPath)
		} else {
			whMName = mutateConfig.Name
			rules = checkRules(mutateConfig.Rules)
			if rules != nil {
				whM, err = buildWebhook(mgr, "custom.admission.control", "/mutate", admissiontypes.WebhookTypeMutating, rules)
			}
		}
	}

	namespace := os.Getenv("POD_NAMESPACE")
	if len(namespace) == 0 {
		namespace = "default"
	}
	secretName := os.Getenv("SECRET_NAME")
	if len(secretName) == 0 {
		secretName = "admission-webhook-certs"
	}
	serviceName := os.Getenv("WEBHOOK_SERVICE_NAME")
	if len(serviceName) == 0 {
		serviceName = "admission-control-service"
	}
	certDir := os.Getenv("CERT_DIR")
	if len(certDir) == 0 {
		certDir = "/tmp/cert"
	}
	var webhookServerName = "custom-admission-webhook"
	logt.Info("webhook server settings", "name", webhookServerName, "namespace", namespace,
		"secret name", secretName, "service name", serviceName, "cert dir", certDir)

	svr, err := webhook.NewServer(webhookServerName, mgr, webhook.ServerOptions{
		Port:    8888,
		CertDir: certDir,
		BootstrapOptions: &webhook.BootstrapOptions{
			Secret: &k8types.NamespacedName{
				Namespace: namespace,
				Name:      secretName,
			},
			Service: &webhook.Service{
				Namespace: namespace,
				Name:      serviceName,
				// Selectors should select the pods that runs this webhook server.
				Selectors: map[string]string{
					"control-plane":           "controller-manager",
					"controller-tools.k8s.io": "1.0",
					"app.kubernetes.io/name":  "admission-control",
				},
			},
		},
	})
	if err != nil {
		logt.Error(err, "failed to start webhook server")
		return err
	}

	svr.ValidatingWebhookConfigName = validatingWebhookName
	svr.MutatingWebhookConfigName = mutatingWebhookName
	if whVName != "" {
		svr.ValidatingWebhookConfigName = whVName
	}
	if whMName != "" {
		svr.MutatingWebhookConfigName = whMName
	}
	logt.Info("webhook names", "mutate", svr.MutatingWebhookConfigName, "validate", svr.ValidatingWebhookConfigName)

	if whV != nil && whM != nil {
		err = svr.Register(whV, whM)
		logt.Info("registered validation webhook & mutation webhook")
	} else if whV != nil && whM == nil {
		err = svr.Register(whV)
		logt.Info("registered validation webhook")
	} else if whV == nil && whM != nil {
		err = svr.Register(whM)
		logt.Info("registered mutation webhook")
	} else {
		logt.Info("registered no webhook")
		err = fmt.Errorf("no webhook configuration is available for registration")
	}
	if err != nil {
		logt.Error(err, "failed to register webhooks")
		return err
	}
	logt.Info("webhook server started !!! ")
	return nil
}

// checkRules validates if all required rule parameters are specified, if not removes the invalid rules
func checkRules(rules []admissionregistrationv1beta1.RuleWithOperations) []admissionregistrationv1beta1.RuleWithOperations {
	var validRules []admissionregistrationv1beta1.RuleWithOperations
	for _, rule := range rules { //verify input rules and remove invalid ones
		if len(rule.Operations) != 0 && len(rule.APIGroups) != 0 && len(rule.APIVersions) != 0 && len(rule.Resources) != 0 {
			validRules = append(validRules, rule)
		} else {
			logt.Info("Invalid rule removed", "rule", rule)
		}
	}
	return validRules
}

// buildWebhook prepares a WebhookBuilder with webhook configuration.
// ToDo: support more than 5 rules per webhook config
func buildWebhook(mgr manager.Manager, name string, path string, whType admissiontypes.WebhookType,
	rules []admissionregistrationv1beta1.RuleWithOperations) (*admission.Webhook, error) {

	logt.Info("buildWebhook", "path", path, "rules", rules)
	number := len(rules)
	if number == 0 {
		return nil, fmt.Errorf("no rules are provided")
	}

	whBuilder := builder.NewWebhookBuilder().
		Name(name).
		Path(path).
		WithManager(mgr)
	if whType == admissiontypes.WebhookTypeMutating {
		whBuilder.
			Mutating().
			Handlers(&AdmissionMutator{})
	} else if whType == admissiontypes.WebhookTypeValidating {
		whBuilder.
			Validating().
			Handlers(&AdmissionValidator{})
	} else {
		return nil, fmt.Errorf("unknow webhook type")
	}
	switch number {
	case 1:
		whBuilder.Rules(rules[0])
	case 2:
		whBuilder.Rules(rules[0], rules[1])
	case 3:
		whBuilder.Rules(rules[0], rules[1], rules[2])
	case 4:
		whBuilder.Rules(rules[0], rules[1], rules[2], rules[3])
	default:
		whBuilder.Rules(rules[0], rules[1], rules[2], rules[3], rules[4])
	}
	wh, err := whBuilder.Build()
	if err != nil {
		logt.Error(err, "failed to create Mutating webhook ")
		return nil, err
	}
	return wh, nil
}
