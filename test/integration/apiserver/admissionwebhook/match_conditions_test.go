package admissionwebhook

import (
	"context"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	genericfeatures "k8s.io/apiserver/pkg/features"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	featuregatetesting "k8s.io/component-base/featuregate/testing"
	apiservertesting "k8s.io/kubernetes/cmd/kube-apiserver/app/testing"
	"k8s.io/kubernetes/test/integration/framework"
)

// Test_MatchConditions tests a ValidatingWebhookConfigurations that validates different cases of matchCondition fields
// a webhook is created with failPolicy fail to force all matching conditions to fail
func Test_MatchConditions_ValidatingWebhookConfiguration(t *testing.T) {
	testcases := []struct {
		name              string
		validatingWebhook *admissionregistrationv1.ValidatingWebhookConfiguration
		namespace         *v1.Namespace
		expectError       bool
	}{
		{
			name: "namespace name contains suffix exempt from vwh",
			validatingWebhook: brokenValidatingWebhookConfiguration("test", []admissionregistrationv1.MatchCondition{
				{
					Name:       "test1",
					Expression: "object.metadata.name.endsWith('k8s')",
				},
			}),
			namespace: &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-k8s",
				},
			},
			expectError: false,
		},
		//{
		//	name: "namespace name does NOT contain suffix enforced by validating admission policy using, object metadata fields",
		//	policy: withValidations([]admissionregistrationv1alpha1.Validation{
		//		{
		//			Expression: "object.metadata.name.endsWith('k8s')",
		//		},
		//	}, withFailurePolicy(admissionregistrationv1alpha1.Fail, withNamespaceMatch(makePolicy("validate-namespace-suffix")))),
		//	policyBinding: makeBinding("validate-namespace-suffix-binding", "validate-namespace-suffix", ""),
		//	namespace: &v1.Namespace{
		//		ObjectMeta: metav1.ObjectMeta{
		//			Name: "test-foobar",
		//		},
		//	},
		//	err:           "namespaces \"test-foobar\" is forbidden: ValidatingAdmissionPolicy 'validate-namespace-suffix' with binding 'validate-namespace-suffix-binding' denied request: failed expression: object.metadata.name.endsWith('k8s')",
		//	failureReason: metav1.StatusReasonInvalid,
		//},
		//{
		//	name: "namespace name contains suffix enforced by validating admission policy, using request field",
		//	policy: withValidations([]admissionregistrationv1alpha1.Validation{
		//		{
		//			Expression: "request.name.endsWith('k8s')",
		//		},
		//	}, withFailurePolicy(admissionregistrationv1alpha1.Fail, withNamespaceMatch(makePolicy("validate-namespace-suffix")))),
		//	policyBinding: makeBinding("validate-namespace-suffix-binding", "validate-namespace-suffix", ""),
		//	namespace: &v1.Namespace{
		//		ObjectMeta: metav1.ObjectMeta{
		//			Name: "test-k8s",
		//		},
		//	},
		//	err: "",
		//},
		//{
		//	name: "namespace name does NOT contains suffix enforced by validating admission policy, using request field",
		//	policy: withValidations([]admissionregistrationv1alpha1.Validation{
		//		{
		//			Expression: "request.name.endsWith('k8s')",
		//		},
		//	}, withFailurePolicy(admissionregistrationv1alpha1.Fail, withNamespaceMatch(makePolicy("validate-namespace-suffix")))),
		//	policyBinding: makeBinding("validate-namespace-suffix-binding", "validate-namespace-suffix", ""),
		//	namespace: &v1.Namespace{
		//		ObjectMeta: metav1.ObjectMeta{
		//			Name: "test-k8s",
		//		},
		//	},
		//	err: "",
		//},
		//{
		//	name: "runtime error when validating namespace match conditions",
		//	policy: withValidations([]admissionregistrationv1alpha1.Validation{
		//		{
		//			Expression: "object.nonExistentProperty == 'someval'",
		//		},
		//	}, withFailurePolicy(admissionregistrationv1alpha1.Ignore, withNamespaceMatch(makePolicy("validate-namespace-suffix")))),
		//	policyBinding: makeBinding("validate-namespace-suffix-binding", "validate-namespace-suffix", ""),
		//	namespace: &v1.Namespace{
		//		ObjectMeta: metav1.ObjectMeta{
		//			Name: "test-k8s",
		//		},
		//	},
		//	err: "",
		//},
		//{
		//	name: "runtime error due to unguarded params",
		//	policy: withValidations([]admissionregistrationv1alpha1.Validation{
		//		{
		//			Expression: "object.metadata.name.startsWith(params.metadata.name)",
		//		},
		//	}, withParams(configParamKind(), withFailurePolicy(admissionregistrationv1alpha1.Fail, withNamespaceMatch(makePolicy("validate-namespace-suffix"))))),
		//	policyBinding: makeBinding("validate-namespace-suffix-binding", "validate-namespace-suffix", ""),
		//	namespace: &v1.Namespace{
		//		ObjectMeta: metav1.ObjectMeta{
		//			Name: "test-k8s",
		//		},
		//	},
		//	err:           "namespaces \"test-k8s\" is forbidden: ValidatingAdmissionPolicy 'validate-namespace-suffix' with binding 'validate-namespace-suffix-binding' denied request: expression 'object.metadata.name.startsWith(params.metadata.name)' resulted in error: no such key: metadata",
		//	failureReason: metav1.StatusReasonInvalid,
		//},
		//{
		//	name: "with check against unguarded params using has()",
		//	policy: withValidations([]admissionregistrationv1alpha1.Validation{
		//		{
		//			Expression: "has(params.metadata) && has(params.metadata.name) && object.metadata.name.endsWith(params.metadata.name)",
		//		},
		//	}, withParams(configParamKind(), withFailurePolicy(admissionregistrationv1alpha1.Fail, withNamespaceMatch(makePolicy("validate-namespace-suffix"))))),
		//	policyBinding: makeBinding("validate-namespace-suffix-binding", "validate-namespace-suffix", ""),
		//	namespace: &v1.Namespace{
		//		ObjectMeta: metav1.ObjectMeta{
		//			Name: "test-k8s",
		//		},
		//	},
		//	err:           "namespaces \"test-k8s\" is forbidden: ValidatingAdmissionPolicy 'validate-namespace-suffix' with binding 'validate-namespace-suffix-binding' denied request: expression 'has(params.metadata) && has(params.metadata.name) && object.metadata.name.endsWith(params.metadata.name)' resulted in error: invalid type for field selection.",
		//	failureReason: metav1.StatusReasonInvalid,
		//},
		//{
		//	name: "with check against null params",
		//	policy: withValidations([]admissionregistrationv1alpha1.Validation{
		//		{
		//			Expression: "(params != null && object.metadata.name.endsWith(params.metadata.name))",
		//		},
		//	}, withParams(configParamKind(), withFailurePolicy(admissionregistrationv1alpha1.Fail, withNamespaceMatch(makePolicy("validate-namespace-suffix"))))),
		//	policyBinding: makeBinding("validate-namespace-suffix-binding", "validate-namespace-suffix", ""),
		//	namespace: &v1.Namespace{
		//		ObjectMeta: metav1.ObjectMeta{
		//			Name: "test-k8s",
		//		},
		//	},
		//	err:           "namespaces \"test-k8s\" is forbidden: ValidatingAdmissionPolicy 'validate-namespace-suffix' with binding 'validate-namespace-suffix-binding' denied request: failed expression: (params != null && object.metadata.name.endsWith(params.metadata.name))",
		//	failureReason: metav1.StatusReasonInvalid,
		//},
		//{
		//	name: "with check against unguarded params using has() and default check",
		//	policy: withValidations([]admissionregistrationv1alpha1.Validation{
		//		{
		//			Expression: "(has(params.metadata) && has(params.metadata.name) && object.metadata.name.startsWith(params.metadata.name)) || object.metadata.name.endsWith('k8s')",
		//		},
		//	}, withParams(configParamKind(), withFailurePolicy(admissionregistrationv1alpha1.Fail, withNamespaceMatch(makePolicy("validate-namespace-suffix"))))),
		//	policyBinding: makeBinding("validate-namespace-suffix-binding", "validate-namespace-suffix", ""),
		//	namespace: &v1.Namespace{
		//		ObjectMeta: metav1.ObjectMeta{
		//			Name: "test-k8s",
		//		},
		//	},
		//	err: "",
		//},
		//{
		//	name: "with check against null params and default check",
		//	policy: withValidations([]admissionregistrationv1alpha1.Validation{
		//		{
		//			Expression: "(params != null && object.metadata.name.startsWith(params.metadata.name)) || object.metadata.name.endsWith('k8s')",
		//		},
		//	}, withParams(configParamKind(), withFailurePolicy(admissionregistrationv1alpha1.Fail, withNamespaceMatch(makePolicy("validate-namespace-suffix"))))),
		//	policyBinding: makeBinding("validate-namespace-suffix-binding", "validate-namespace-suffix", ""),
		//	namespace: &v1.Namespace{
		//		ObjectMeta: metav1.ObjectMeta{
		//			Name: "test-k8s",
		//		},
		//	},
		//	err: "",
		//},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			defer featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, genericfeatures.AdmissionWebhookMatchConditions, true)()
			server, err := apiservertesting.StartTestServer(t, nil, nil, framework.SharedEtcd())
			if err != nil {
				t.Fatal(err)
			}
			defer server.TearDownFn()

			config := server.ClientConfig

			client, err := clientset.NewForConfig(config)
			if err != nil {
				t.Fatal(err)
			}
			//webhook := brokenValidatingWebhookConfiguration("test", nil)
			//if _, err := client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(context.TODO(), webhook, metav1.CreateOptions{}); err != nil {
			//	t.Fatal(err)
			//}
			if err := createAndWaitReady(t, client, testcase.validatingWebhook, nil); err != nil {
				t.Fatal(err)
			}

			_, err = client.CoreV1().Namespaces().Create(context.TODO(), testcase.namespace, metav1.CreateOptions{})

		})
	}
}

func createAndWaitReady(t *testing.T, client *clientset.Clientset, webhookconfig *admissionregistrationv1.ValidatingWebhookConfiguration, matchLabels map[string]string) error {
	marker := &v1.Endpoints{ObjectMeta: metav1.ObjectMeta{Name: "test-marker", Namespace: "default", Labels: matchLabels}}
	defer func() {
		err := client.CoreV1().Endpoints("default").Delete(context.TODO(), marker.Name, metav1.DeleteOptions{})
		if err != nil {
			t.Logf("error deleting marker: %v", err)
		}
	}()
	marker, err := client.CoreV1().Endpoints("default").Create(context.TODO(), marker, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(context.TODO(), webhookconfig, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	if waitErr := wait.PollImmediate(time.Millisecond*5, wait.ForeverTestTimeout, func() (bool, error) {
		resp, err := client.CoreV1().Endpoints("default").Patch(context.TODO(), marker.Name, types.JSONPatchType, []byte("[]"), metav1.PatchOptions{})
		if err != nil && strings.Contains(err.Error(), "marker denied; policy is ready") {
			return true, nil
		} else if err != nil && strings.Contains(err.Error(), "not yet synced to use for admission") {
			t.Logf("waiting for webhook config to be ready. Marker: %v. Admission not synced yet: %v", marker, err)
			return false, nil
		} else {
			t.Logf("waiting for webhook config to be ready. Marker: %v, Last marker patch error: %v. Resp: %v", marker, err, resp)
			return false, err
		}
	}); waitErr != nil {
		return waitErr
	}
	return nil
}

func brokenValidatingWebhookConfiguration(name string, matchConditions []admissionregistrationv1.MatchCondition) *admissionregistrationv1.ValidatingWebhookConfiguration {
	var path string
	failurePolicy := admissionregistrationv1.Fail
	return &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{
			{
				Name: "broken-webhook.k8s.io",
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.OperationAll},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{"*"},
						APIVersions: []string{"*"},
						Resources:   []string{"*"},
					},
				}},
				// This client config references a non existent service
				// so it should always fail.
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: "default",
						Name:      "invalid-webhook-service",
						Path:      &path,
					},
					CABundle: nil,
				},
				FailurePolicy:           &failurePolicy,
				SideEffects:             &noSideEffects,
				AdmissionReviewVersions: []string{"v1"},
				MatchConditions:         matchConditions,
			},
		},
	}
}
