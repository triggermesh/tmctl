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
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type CRD struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name string `yaml:"name"`
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

type release struct {
	TagName string `json:"tag_name"`
}

const ghLatestRelease = "https://api.github.com/repos/triggermesh/triggermesh/releases/latest"

func latest() (string, error) {
	r, err := http.Get(ghLatestRelease)
	if err != nil {
		return "", fmt.Errorf("cannot detect latest release tag: %w", err)
	}
	defer r.Body.Close()
	var j release

	return j.TagName, json.NewDecoder(r.Body).Decode(&j)
}

func Fetch(configDir string) (string, error) {
	var err error
	version := viper.GetString("triggermesh.version")
	if version == "latest" {
		if version, err = latest(); err != nil {
			return "", fmt.Errorf("cannot fetch latest CRD: %w", err)
		}
	}
	url := strings.ReplaceAll(viper.GetString("triggermesh.crd"), "${VERSION}", version)
	crdDir := path.Join(configDir, "crd", version)
	if err := os.MkdirAll(crdDir, os.ModePerm); err != nil {
		return "", err
	}
	crdFile := path.Join(crdDir, "crd.yaml")
	if _, err := os.Stat(crdFile); err == nil {
		return crdFile, nil
	}
	log.Println("Fetching CRD")
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
	_, err = io.Copy(out, resp.Body)
	return crdFile, err
}

func GetResource(resource, path string) (CRD, error) {
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
