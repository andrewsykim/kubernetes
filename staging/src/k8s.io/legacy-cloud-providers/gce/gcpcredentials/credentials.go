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

package gcpcredentials

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/klog/v2"
)

const (
	metadataURL               = "http://metadata.google.internal./computeMetadata/v1/"
	metadataAttributes        = metadataURL + "instance/attributes/"
	dockerConfigKey           = metadataAttributes + "google-dockercfg"
	dockerConfigURLKey        = metadataAttributes + "google-dockercfg-url"
	serviceAccounts           = metadataURL + "instance/service-accounts/"
	metadataScopes            = metadataURL + "instance/service-accounts/default/scopes"
	metadataToken             = metadataURL + "instance/service-accounts/default/token"
	metadataEmail             = metadataURL + "instance/service-accounts/default/email"
	storageScopePrefix        = "https://www.googleapis.com/auth/devstorage"
	cloudPlatformScopePrefix  = "https://www.googleapis.com/auth/cloud-platform"
	defaultServiceAccount     = "default/"
	metadataHTTPClientTimeout = time.Second * 10
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

var httpClient = &http.Client{
	Transport: utilnet.SetTransportDefaults(&http.Transport{}),
	Timeout:   metadataHTTPClientTimeout,
}

// A DockerConfigProvider that reads its configuration from Google
// Compute Engine metadata.
type metadataProvider struct {
	Client *http.Client
}

// A DockerConfigProvider that reads its configuration from a specific
// Google Compute Engine metadata key: 'google-dockercfg'.
type dockerConfigKeyProvider struct {
	metadataProvider
}

// A DockerConfigProvider that reads its configuration from a URL read from
// a specific Google Compute Engine metadata key: 'google-dockercfg-url'.
type dockerConfigURLKeyProvider struct {
	metadataProvider
}

// A DockerConfigProvider that provides a dockercfg with:
//    Username: "_token"
//    Password: "{access token from metadata}"
type containerRegistryProvider struct {
	metadataProvider
}

// DockerConfigJSON represents ~/.docker/config.json file info
// see https://github.com/docker/docker/pull/12009
type DockerConfigJSON struct {
	Auths DockerConfig `json:"auths"`
	// +optional
	HTTPHeaders map[string]string `json:"HttpHeaders,omitempty"`
}

// DockerConfig represents the config file used by the docker CLI.
// This config that represents the credentials that should be used
// when pulling images from specific image repositories.
type DockerConfig map[string]DockerConfigEntry

// DockerConfigEntry wraps a docker config as a entry
type DockerConfigEntry struct {
	Username string
	Password string
	Email    string
}

// Returns true if it finds a local GCE VM.
// Looks at a product file that is an undocumented API.
func OnGCEVM() bool {
	var name string

	if runtime.GOOS == "windows" {
		data, err := exec.Command("wmic", "computersystem", "get", "model").Output()
		if err != nil {
			return false
		}
		fields := strings.Split(strings.TrimSpace(string(data)), "\r\n")
		if len(fields) != 2 {
			klog.V(2).Infof("Received unexpected value retrieving system model: %q", string(data))
			return false
		}
		name = fields[1]
	} else {
		data, err := ioutil.ReadFile(gceProductNameFile)
		if err != nil {
			klog.V(2).Infof("Error while reading product_name: %v", err)
			return false
		}
		name = strings.TrimSpace(string(data))
	}
	return name == "Google" || name == "Google Compute Engine"
}

func HasStorageScope() bool {
	// Given that we are on GCE, we should keep retrying until the metadata server responds.
	value := runWithBackoff(func() ([]byte, error) {
		value, err := readURL(serviceAccounts, httpClient, metadataHeader)
		if err != nil {
			klog.V(2).Infof("Failed to Get service accounts from gce metadata server: %v", err)
		}
		return value, err
	})

	// We expect the service account to return a list of account directories separated by newlines, e.g.,
	//   sv-account-name1/
	//   sv-account-name2/
	// ref: https://cloud.google.com/compute/docs/storing-retrieving-metadata
	defaultServiceAccountExists := false
	for _, sa := range strings.Split(string(value), "\n") {
		if strings.TrimSpace(sa) == defaultServiceAccount {
			defaultServiceAccountExists = true
			break
		}
	}
	if !defaultServiceAccountExists {
		klog.V(2).Infof("'default' service account does not exist. Found following service accounts: %q", string(value))
		return false
	}
	url := metadataScopes + "?alt=json"
	value = runWithBackoff(func() ([]byte, error) {
		value, err := readURL(url, httpClient, metadataHeader)
		if err != nil {
			klog.V(2).Infof("Failed to Get scopes in default service account from gce metadata server: %v", err)
		}
		return value, err
	})
	var scopes []string
	if err := json.Unmarshal(value, &scopes); err != nil {
		klog.Errorf("Failed to unmarshal scopes: %v", err)
		return false
	}
	for _, v := range scopes {
		// cloudPlatformScope implies storage scope.
		if strings.HasPrefix(v, storageScopePrefix) || strings.HasPrefix(v, cloudPlatformScopePrefix) {
			return true
		}
	}
	klog.Warningf("Google container registry is disabled, no storage scope is available: %s", value)
	return false

}

// Provide implements DockerConfigProvider
func ProvideDockerConfigKey(image string) DockerConfig {
	// Read the contents of the google-dockercfg metadata key and
	// parse them as an alternate .dockercfg
	if cfg, err := ReadDockerConfigFileFromURL(dockerConfigKey, httpClient, metadataHeader); err != nil {
		klog.Errorf("while reading 'google-dockercfg' metadata: %v", err)
	} else {
		return cfg
	}

	return DockerConfig{}
}

// Provide implements DockerConfigProvider
func ProvideDockerConfigURLKey(image string) DockerConfig {
	// Read the contents of the google-dockercfg-url key and load a .dockercfg from there
	if url, err := readURL(dockerConfigURLKey, httpClient, metadataHeader); err != nil {
		klog.Errorf("while reading 'google-dockercfg-url' metadata: %v", err)
	} else {
		if strings.HasPrefix(string(url), "http") {
			if cfg, err := ReadDockerConfigFileFromURL(string(url), httpClient, nil); err != nil {
				klog.Errorf("while reading 'google-dockercfg-url'-specified url: %s, %v", string(url), err)
			} else {
				return cfg
			}
		} else {
			// TODO(mattmoor): support reading alternate scheme URLs (e.g. gs:// or s3://)
			klog.Errorf("Unsupported URL scheme: %s", string(url))
		}
	}

	return DockerConfig{}
}

// tokenBlob is used to decode the JSON blob containing an access token
// that is returned by GCE metadata.
type tokenBlob struct {
	AccessToken string `json:"access_token"`
}

// Provide implements DockerConfigProvider
func ProvideContainerRegistry(image string) DockerConfig {
	cfg := DockerConfig{}

	tokenJSONBlob, err := readURL(metadataToken, httpClient, metadataHeader)
	if err != nil {
		klog.Errorf("while reading access token endpoint: %v", err)
		return cfg
	}

	email, err := readURL(metadataEmail, httpClient, metadataHeader)
	if err != nil {
		klog.Errorf("while reading email endpoint: %v", err)
		return cfg
	}

	var parsedBlob tokenBlob
	if err := json.Unmarshal([]byte(tokenJSONBlob), &parsedBlob); err != nil {
		klog.Errorf("while parsing json blob %s: %v", tokenJSONBlob, err)
		return cfg
	}

	entry := DockerConfigEntry{
		Username: "_token",
		Password: parsedBlob.AccessToken,
		Email:    string(email),
	}

	// Add our entry for each of the supported container registry URLs
	for _, k := range containerRegistryUrls {
		cfg[k] = entry
	}
	return cfg
}

// readURL read contents from given url
func readURL(url string, client *http.Client, header *http.Header) (body []byte, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if header != nil {
		req.Header = *header
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		klog.V(2).Infof("body of failing http response: %v", resp.Body)
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			URL:        url,
		}
	}

	limitedReader := &io.LimitedReader{R: resp.Body, N: maxReadLength}
	contents, err := ioutil.ReadAll(limitedReader)
	if err != nil {
		return nil, err
	}

	if limitedReader.N <= 0 {
		return nil, errors.New("the read limit is reached")
	}

	return contents, nil
}

// ReadDockerConfigFileFromURL read a docker config file from the given url
func ReadDockerConfigFileFromURL(url string, client *http.Client, header *http.Header) (cfg DockerConfig, err error) {
	if contents, err := readURL(url, client, header); err == nil {
		return readDockerConfigFileFromBytes(contents)
	}

	return nil, err
}

func readDockerConfigFileFromBytes(contents []byte) (cfg DockerConfig, err error) {
	if err = json.Unmarshal(contents, &cfg); err != nil {
		return nil, errors.New("error occurred while trying to unmarshal json")
	}
	return
}

// runWithBackoff runs input function `f` with an exponential backoff.
// Note that this method can block indefinitely.
func runWithBackoff(f func() ([]byte, error)) []byte {
	var backoff = 100 * time.Millisecond
	const maxBackoff = time.Minute
	for {
		value, err := f()
		if err == nil {
			return value
		}
		time.Sleep(backoff)
		backoff = backoff * 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}
