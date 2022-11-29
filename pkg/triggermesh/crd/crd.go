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
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/triggermesh/tmctl/pkg/log"
)

const (
	crdsURL = "https://github.com/triggermesh/triggermesh/releases/download/$VERSION/triggermesh-crds.yaml"
)

type CRD struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name        string `yaml:"name"`
		Annotations struct {
			EventTypes string `yaml:"registry.knative.dev/eventTypes"`
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
	Type string `json:"type"`
}

func Fetch(configDir, version string) (string, error) {
	var err error
	url := strings.ReplaceAll(crdsURL, "$VERSION", version)
	crdDir := filepath.Join(configDir, "crd", version)
	if err := os.MkdirAll(crdDir, os.ModePerm); err != nil {
		return "", err
	}
	crdFile := filepath.Join(crdDir, "crd.yaml")
	if _, err := os.Stat(crdFile); err == nil {
		return crdFile, nil
	}
	log.Printf("Fetching %s CRD", version)
	out, err := os.Create(crdFile)
	if err != nil {
		return "", err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("CRD request failed: %s", resp.Status)
	}
	_, err = io.Copy(out, resp.Body)
	return crdFile, err
}

func GetResourceCRD(resource, path string) (CRD, error) {
	crds, err := readFile(path)
	if err != nil {
		return CRD{}, err
	}
	crd, ok := crds[strings.ToLower(resource)]
	if !ok {
		return CRD{}, fmt.Errorf("CRD for resource %q does not exist", resource)
	}
	return crd, nil
}

func readFile(path string) (map[string]CRD, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	var crds []CRD
	decoder := yaml.NewDecoder(f)
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

func ListSources(crdFile string) ([]string, error) {
	crds, err := readFile(crdFile)
	if err != nil {
		return []string{}, err
	}
	var result []string
	for k, crd := range crds {
		if crd.Spec.Group == "sources.triggermesh.io" {
			result = append(result, strings.TrimSuffix(k, "source"))
		}
	}
	sort.Strings(result)
	return result, nil
}

func ListTargets(crdFile string) ([]string, error) {
	crds, err := readFile(crdFile)
	if err != nil {
		return []string{}, err
	}
	var result []string
	for k, crd := range crds {
		if crd.Spec.Group == "targets.triggermesh.io" {
			result = append(result, strings.TrimSuffix(k, "target"))
		}
	}
	sort.Strings(result)
	return result, nil
}
