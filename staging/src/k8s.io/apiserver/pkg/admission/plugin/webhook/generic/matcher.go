/*
Copyright 2023 The Kubernetes Authors.

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

package generic

import (
	celtypes "github.com/google/cel-go/common/types"

	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/admission/plugin/cel"
	"k8s.io/klog/v2"
)

// matcher implements the Matcher interface
type matcher struct {
	filter     cel.Filter
}

func NewMatcher(filter cel.Filter) Matcher {
	return &matcher{
		filter:     filter,
	}
}

// Match takes a list of Evaluation converts them into actionable a decision of whether they match or not.
// If no match returns name of failed condition
func (v *matcher) Match(versionedAttr *admission.VersionedAttributes) (bool, string, error) {

	evalResults, err := v.filter.ForInput(versionedAttr, nil, cel.CreateAdmissionRequest(versionedAttr.Attributes))
	if err != nil {
		// on error ignore the match condition, don't apply matching
		return true, "", err
	}

	for _, evalResult := range evalResults {
		if evalResult.EvalResult == celtypes.False {
			matchCondition, ok := evalResult.ExpressionAccessor.(*MatchCondition)
			if !ok {
				klog.Error("Invalid type conversion to MatchCondition")
				continue
			}
			return false, matchCondition.Name, nil
		}
	}
	// if no failures or empty list return true
	return true, "", nil
}
