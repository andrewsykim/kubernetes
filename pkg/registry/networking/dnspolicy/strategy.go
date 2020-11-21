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

package dnspolicy

import (
	"context"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/storage/names"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	"k8s.io/kubernetes/pkg/apis/networking"
	"k8s.io/kubernetes/pkg/apis/networking/validation"
)

// dnsPolicyStrategy implements verification and REST strategy logic for DNSPolicy objects.
type dnsPolicyStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

// Strategy is the default logic that applies when creating and updating DNSPolicy objects.
var Strategy = dnsPolicyStrategy{legacyscheme.Scheme, names.SimpleNameGenerator}

// NamespaceScoped returns true because all DNSPolicies need to be within a namespace.
func (dnsPolicyStrategy) NamespaceScoped() bool {
	return true
}

func (dnsPolicyStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	dnsPolicy := obj.(*networking.DNSPolicy)
	dnsPolicy.Generation = 1
}

func (dnsPolicyStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newDNSPolicy := obj.(*networking.DNSPolicy)
	oldDNSPolicy := old.(*networking.DNSPolicy)

	if !reflect.DeepEqual(oldDNSPolicy, newDNSPolicy) {
		newDNSPolicy.Generation = oldDNSPolicy.Generation + 1
	}
}

// Validate validates a new DNSPolicy.
func (dnsPolicyStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	dnsPolicy := obj.(*networking.DNSPolicy)
	return validation.ValidateDNSPolicy(dnsPolicy)
}

// Canonicalize normalizes the object after validation.
func (dnsPolicyStrategy) Canonicalize(obj runtime.Object) {}

// AllowCreateOnUpdate is false for DNSPolicy; this means POST is needed to create one.
func (dnsPolicyStrategy) AllowCreateOnUpdate() bool {
	return false
}

// ValidateUpdate validates updates to DNSPolicy.
func (dnsPolicyStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newDNSPolicy := obj.(*networking.DNSPolicy)
	oldDNSPolicy := old.(*networking.DNSPolicy)

	validationErrs := validation.ValidateDNSPolicy(newDNSPolicy)
	validationUpdateErrs := validation.ValidateDNSPolicyUpdate(newDNSPolicy, oldDNSPolicy)

	return append(validationErrs, validationUpdateErrs...)
}

// AllowUnconditionalUpdate is the default update policy for DNSPolicy objects
func (dnsPolicyStrategy) AllowUnconditionalUpdate() bool {
	return true
}
