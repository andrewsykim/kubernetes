/*
Copyright 2020 The Kubernetes Authors.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:prerelease-lifecycle-gen:introduced=1.20

// DNSPolicy describes what domains are allowed to be resolved by a set of Pods
type DNSPolicy struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Specification of the desired behavior for this DNSPolicy.
	// +optional
	Spec DNSPolicySpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

// DNSPolicySpec provides specification for a DNSPolicy
type DNSPolicySpec struct {
	// podSelector selects pods to which this DNSPolicy object applies. Any pod selected
	// by this policy will only be allowed to resolve domains in the allowedDomains list.
	// Enforcement of this policy against a pod means that the pod can only resolve domains
	// in allowedDomains.
	//
	// An empty podSelector means that this policy should apply to all pods in the namespace.
	// Host network pods are excluded from this policy.
	PodSelector metav1.LabelSelector `json:"podSelector" protobuf:"bytes,1,opt,name=podSelector"`

	// allowedDomains is a list of domains that are resolvable for pods selected by podSelector.
	// Only fully qualified domain names (e.g. www.example.com) and wildcard subdomains (e.g. *.example.com)
	// are supported. Wildcards are only allowed on the left-most parts of the domain, so *.*.example.com
	// is allowed, but example.* is not. Matching domains must have the same number of parts, so
	// foo.bar.example.com does not match for *.example.com.
	// +optional
	AllowedDomains []string `json:"allowedDomains,omitempty" protobuf:"bytes,2,rep,name=allowedDomains"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:prerelease-lifecycle-gen:introduced=1.20

// DNSPolicyList is a list of DNSPolicy objects.
type DNSPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// items is a list of DNSPolicy objects.
	Items []DNSPolicy `json:"items" protobuf:"bytes,2,rep,name=items"`
}
