/*
Copyright 2022 TriggerMesh Inc.

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

package crd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/triggermesh/tmctl/pkg/log"
)

const (
	crdsURL         = "https://github.com/triggermesh/triggermesh/releases/download/$VERSION/triggermesh-crds.yaml"
	commitStatusURL = "https://api.github.com/repos/triggermesh/triggermesh/commits/$REF/status"
	ciArtifactsURL  = "https://circleci.com/api/v2/project/gh/triggermesh/triggermesh/$JOB/artifacts"

	ciBuildImageContext = "ci/circleci: publish-image"
)

var semverRegexp = regexp.MustCompile(`^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

type ghJobStatus struct {
	SHA      string `json:"sha"`
	Statuses []struct {
		State     string `json:"state"`
		TargetURL string `json:"target_url"`
		Context   string `json:"context"`
	} `json:"statuses"`
}

type ciArtifacts struct {
	Items []struct {
		URL string `json:"url"`
	} `json:"items"`
}

type CRD struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name        string `yaml:"name"`
		Annotations struct {
			ProducedEventTypes string `yaml:"registry.knative.dev/eventTypes"`
			ConsumedEventTypes string `yaml:"registry.triggermesh.io/acceptedEventTypes"`
		} `yaml:"annotations"`
	} `yaml:"metadata"`
	Spec struct {
		Group string `yaml:"group"`
		Scope string `yaml:"scope"`
		Names struct {
			Kind       string   `yaml:"kind"`
			Plural     string   `yaml:"plural"`
			Categories []string `yaml:"categories"`
		} `yaml:"names"`
		Versions []struct {
			Name         string `yaml:"name"`
			Served       bool   `yaml:"served"`
			Storage      bool   `yaml:"storage"`
			Subresources struct {
				Status struct {
				} `yaml:"status"`
			} `yaml:"subresources"`
			Schema struct {
				OpenAPIV3Schema struct {
					Properties struct {
						Spec map[string]interface{} `yaml:"spec"`
					} `yaml:"properties"`
				} `yaml:"openAPIV3Schema"`
			} `yaml:"schema"`
		} `yaml:"versions"`
	} `yaml:"spec"`
}

type EventTypes []struct {
	Type   string `json:"type"`
	Schema string `json:"schema"`
}

// Fetch downloads the release version of TriggerMesh CRDs for specified version.
func Fetch(configDir, version string) (map[string]CRD, error) {
	crdDir := filepath.Join(configDir, "crd", version)
	crdFile := filepath.Join(crdDir, "crd.yaml")
	if stat, err := os.Stat(crdFile); err == nil && stat.Size() != 0 {
		f, err := os.Open(crdFile)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return Parse(f)
	}
	if err := os.MkdirAll(crdDir, os.ModePerm); err != nil {
		return nil, err
	}
	log.Printf("Fetching %s CRD", version)
	out, err := os.Create(crdFile)
	if err != nil {
		return nil, err
	}
	defer out.Close()

	var resp *http.Response
	if release := semverRegexp.MatchString(version); release {
		if resp, err = http.Get(strings.ReplaceAll(crdsURL, "$VERSION", version)); err != nil {
			return nil, err
		}
	} else {
		if resp, err = getBuildArtifact(version); err != nil {
			return nil, fmt.Errorf("%q build: %w", version, err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CRD request failed: %s", resp.Status)
	}
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(crdFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

func Parse(reader io.ReadCloser) (map[string]CRD, error) {
	var crds []CRD
	decoder := yaml.NewDecoder(reader)
	for {
		c := new(CRD)
		err := decoder.Decode(&c)
		if c == nil {
			continue
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		crds = append(crds, *c)
	}
	result := make(map[string]CRD, len(crds))
	for _, v := range crds {
		result[strings.ToLower(v.Spec.Names.Kind)] = v
	}
	return result, nil
}

// ListSources returns the list of resources of the "source" API group from CRD.
func ListSources(crds map[string]CRD) ([]string, error) {
	// crds, err := parse(crdReader)
	// if err != nil {
	// 	return []string{}, err
	// }
	var result []string
	for k, crd := range crds {
		if crd.Spec.Group == "sources.triggermesh.io" {
			result = append(result, strings.TrimSuffix(k, "source"))
		}
	}
	sort.Strings(result)
	return result, nil
}

// ListTargets returns the list of resources of the "target" API group from CRD.
func ListTargets(crds map[string]CRD) ([]string, error) {
	// crds, err := parse(crdReader)
	// if err != nil {
	// 	return []string{}, err
	// }
	var result []string
	for k, crd := range crds {
		if crd.Spec.Group == "targets.triggermesh.io" {
			result = append(result, strings.TrimSuffix(k, "target"))
		}
	}
	sort.Strings(result)
	return result, nil
}

func getBuildArtifact(ref string) (*http.Response, error) {
	ghCommit, err := http.Get(strings.Replace(commitStatusURL, "$REF", ref, -1))
	if err != nil {
		return nil, err
	}
	defer ghCommit.Body.Close()

	var ghJob ghJobStatus
	if err := json.NewDecoder(ghCommit.Body).Decode(&ghJob); err != nil {
		return nil, err
	}
	ciJobID := ""
	for _, status := range ghJob.Statuses {
		if status.Context == ciBuildImageContext {
			if status.State != "success" {
				return nil, fmt.Errorf("dev build is not ready")
			}
			path := strings.Split(status.TargetURL, "/")
			ciJobID = path[len(path)-1]
			break
		}
	}
	if ciJobID == "" {
		return nil, fmt.Errorf("CI build job not found")
	}

	ciBuild, err := http.Get(strings.Replace(ciArtifactsURL, "$JOB", ciJobID, -1))
	if err != nil {
		return nil, err
	}
	defer ciBuild.Body.Close()

	var artifacts ciArtifacts
	if err := json.NewDecoder(ciBuild.Body).Decode(&artifacts); err != nil {
		return nil, err
	}
	crdURL := ""
	for _, artifact := range artifacts.Items {
		if strings.HasSuffix(artifact.URL, "/triggermesh-crds.yaml") {
			crdURL = artifact.URL
			break
		}
	}
	if crdURL == "" {
		return nil, fmt.Errorf("CRD artifact not found")
	}
	return http.Get(crdURL)
}
