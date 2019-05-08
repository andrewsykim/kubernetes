/*
Copyright 2014 The Kubernetes Authors.

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

package cloudprovider

import (
	"fmt"
)

const (
	ccm = "cloud-controller-manager"
	kcm = "kube-controller-manager"
)

type MigrationConfig interface {
	GetComponent() string
	// set the component name at runtime
	SetComponent(component string) error

	// returns whether to run cloud node controller based on the controller name
	// and whether it should run under a migration lock
	CloudNodeController() (bool, bool)
	// returns whether to run service controller based on the controller name
	// and whether it should run under a migration lock
	ServiceController() (bool, bool)
	// returns whether to run route controller based on the controller name
	// and whether it should run under a migration lock
	RouteController() (bool, bool)
	// returns whether to run node ipam controller based on the controller name
	// and whether it should run under a migration lock
	NodeIPAMController() (bool, bool)
}

type dummyMigrationConfig struct {
	component string
}

func (m *dummyMigrationConfig) GetComponent() string {
	return m.component
}

func (m *dummyMigrationConfig) SetComponent(component string) error {
	if component != ccm && component != kcm {
		return fmt.Errorf("invalid component name %q", component)
	}

	m.component = component
}

func (m *dummyMigrationConfig) CloudNodeController() (bool, bool) {
	// run cloud node controller in all components under migration
	return true, true
}

func (m *dummyMigrationConfig) ServiceController() (bool, bool) {
	// the service controller is fully migrated to CCM
	if m.component == ccm {
		return true, false
	}

	return false, false
}

func (m *dummyMigrationConfig) RouteController() (bool, bool) {
	// route controller should only run in KCM
	if m.component == kcm {
		return true, false
	}

	return false, false
}

func (m *dummyMigrationConfig) NodeIPAMController() (bool, bool) {
	// node ipam controller can run only in ccm
	if m.component == ccm {
		return true, false
	}

	return false, false
}
