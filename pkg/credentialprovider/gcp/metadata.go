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

package gcp

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/credentialprovider"

	"k8s.io/legacy-cloud-providers/gce/gcpcredentials"
)

const (
	metadataURL              = "http://metadata.google.internal./computeMetadata/v1/"
	metadataAttributes       = metadataURL + "instance/attributes/"
	dockerConfigKey          = metadataAttributes + "google-dockercfg"
	dockerConfigURLKey       = metadataAttributes + "google-dockercfg-url"
	serviceAccounts          = metadataURL + "instance/service-accounts/"
	metadataScopes           = metadataURL + "instance/service-accounts/default/scopes"
	metadataToken            = metadataURL + "instance/service-accounts/default/token"
	metadataEmail            = metadataURL + "instance/service-accounts/default/email"
	storageScopePrefix       = "https://www.googleapis.com/auth/devstorage"
	cloudPlatformScopePrefix = "https://www.googleapis.com/auth/cloud-platform"
	defaultServiceAccount    = "default/"
)

// Product file path that contains the cloud service name.
// This is a variable instead of a const to enable testing.
var gceProductNameFile = "/sys/class/dmi/id/product_name"

// For these urls, the parts of the host name can be glob, for example '*.gcr.io" will match
// "foo.gcr.io" and "bar.gcr.io".
var containerRegistryUrls = []string{"container.cloud.google.com", "gcr.io", "*.gcr.io", "*.pkg.dev"}

var metadataHeader = &http.Header{
	"Metadata-Flavor": []string{"Google"},
}

// A DockerConfigProvider that reads its configuration from a specific
// Google Compute Engine metadata key: 'google-dockercfg'.
type dockerConfigKeyProvider struct {
}

// A DockerConfigProvider that reads its configuration from a URL read from
// a specific Google Compute Engine metadata key: 'google-dockercfg-url'.
type dockerConfigURLKeyProvider struct {
}

// A DockerConfigProvider that provides a dockercfg with:
//    Username: "_token"
//    Password: "{access token from metadata}"
type containerRegistryProvider struct {
}

// init registers the various means by which credentials may
// be resolved on GCP.
func init() {
	credentialprovider.RegisterCredentialProvider("google-dockercfg",
		&credentialprovider.CachingDockerConfigProvider{
			Provider: &dockerConfigKeyProvider{
				metadataProvider{},
			},
			Lifetime: 60 * time.Second,
		})

	credentialprovider.RegisterCredentialProvider("google-dockercfg-url",
		&credentialprovider.CachingDockerConfigProvider{
			Provider: &dockerConfigURLKeyProvider{
				metadataProvider{},
			},
			Lifetime: 60 * time.Second,
		})

	credentialprovider.RegisterCredentialProvider("google-container-registry",
		// Never cache this.  The access token is already
		// cached by the metadata service.
		&containerRegistryProvider{
			metadataProvider{},
		})
}

// Returns true if it finds a local GCE VM.
// Looks at a product file that is an undocumented API.
func onGCEVM() bool {
	return gcpcredentials.OnGCEVM()
}

// Enabled implements DockerConfigProvider for all of the Google implementations.
func (g *metadataProvider) Enabled() bool {
	return onGCEVM()
}

// Provide implements DockerConfigProvider
func (g *dockerConfigKeyProvider) Provide(image string) credentialprovider.DockerConfig {
	cfg := credentialprovider.DockerConfig{}
	gcpCfg := gcpcredentials.ProvideDockerConfigKey(image)

	for key, value := range gcpCfg {
		entry := credentialprovder.DockerConfigEntry{
			Username: value.Username,
			Password: value.Password,
			Email:    value.Email,
		}
		cfg[key] = entry
	}

	return cfg
}

// Provide implements DockerConfigProvider
func (g *dockerConfigURLKeyProvider) Provide(image string) credentialprovider.DockerConfig {
	cfg := credentialprovider.DockerConfig{}
	gcpCfg := gcpcredentials.ProvideDockerConfigURLKey(image)

	for key, value := range gcpCfg {
		entry := credentialprovder.DockerConfigEntry{
			Username: value.Username,
			Password: value.Password,
			Email:    value.Email,
		}
		cfg[key] = entry
	}

	return cfg
}

// Enabled implements a special metadata-based check, which verifies the
// storage scope is available on the GCE VM.
// If running on a GCE VM, check if 'default' service account exists.
// If it does not exist, assume that registry is not enabled.
// If default service account exists, check if relevant scopes exist in the default service account.
// The metadata service can become temporarily inaccesible. Hence all requests to the metadata
// service will be retried until the metadata server returns a `200`.
// It is expected that "http://metadata.google.internal./computeMetadata/v1/instance/service-accounts/" will return a `200`
// and "http://metadata.google.internal./computeMetadata/v1/instance/service-accounts/default/scopes" will also return `200`.
// More information on metadata service can be found here - https://cloud.google.com/compute/docs/storing-retrieving-metadata
func (g *containerRegistryProvider) Enabled() bool {
	if !gcpcredentials.OnGCEVM() {
		return false
	}

	return gcpcredentials.HasStorageScope()
}

// Provide implements DockerConfigProvider
func (g *containerRegistryProvider) Provide(image string) credentialprovider.DockerConfig {
	cfg := credentialprovider.DockerConfig{}
	gcpCfg := gcpcredentials.ProvideContainerRegistry(image)

	for key, value := range gcpCfg {
		entry := credentialprovder.DockerConfigEntry{
			Username: value.Username,
			Password: value.Password,
			Email:    value.Email,
		}
		cfg[key] = entry
	}

	return cfg
}
