/*
Copyright 2023 TriggerMesh Inc.

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

package load

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/triggermesh/tmctl/cmd/describe"
	cliconfig "github.com/triggermesh/tmctl/pkg/config"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

func Import(contextName, from string, config *cliconfig.Config, crd map[string]crd.CRD) error {
	m, err := getManifest(from)
	if err != nil {
		return fmt.Errorf("manifest %q: %w", from, err)
	}

	// create broker first
	for i, object := range m.Objects {
		if object.Kind != tmbroker.BrokerKind {
			continue
		}
		if contextName == "" {
			contextName = object.Metadata.Name
		}
		m.Objects[i].Metadata.Name = contextName
		if _, err := tmbroker.CreateBrokerConfig(config.ConfigHome, contextName); err != nil {
			return fmt.Errorf("creating broker object: %w", err)
		}
		if _, err := tmbroker.New(contextName, config.Triggermesh.Broker); err != nil {
			return fmt.Errorf("creating broker object: %w", err)
		}
		break
	}

	m.Path = filepath.Join(config.ConfigHome, contextName, triggermesh.ManifestFile)

	// insert user input, update context name
	for i, object := range m.Objects {
		m.Objects[i].Metadata.Labels[triggermesh.ContextLabel] = contextName
		component, err := components.GetObject(object.Metadata.Name, config, m, crd)
		if err != nil {
			return err
		}
		filledSpec, err := parseUserInputTags(component.GetName(), component.GetKind(), component.GetSpec())
		if err != nil {
			return err
		}
		component.SetSpec(filledSpec)
		newObj, err := component.AsK8sObject()
		if err != nil {
			return err
		}
		m.Objects[i] = newObj
	}

	if err := m.Write(); err != nil {
		return err
	}

	// write broker config
	for _, object := range m.Objects {
		if object.Kind != tmbroker.TriggerKind {
			continue
		}
		trigger, err := components.GetObject(object.Metadata.Name, config, m, crd)
		if err != nil {
			return err
		}
		if err := trigger.(*tmbroker.Trigger).WriteLocalConfig(); err != nil {
			return err
		}
	}

	descr := &describe.CliOptions{
		Config:   config,
		Manifest: m,
		CRD:      crd,
	}
	_ = descr.Describe()

	log.Printf("Switching context to %q", contextName)
	cliconfig.Set("context", contextName)
	log.Println("Done")
	return nil
}

func getManifest(from string) (*manifest.Manifest, error) {
	_, err := os.Stat(from)
	if os.IsNotExist(err) {
		tempPath, err := fetch(from)
		if err != nil {
			return nil, err
		}
		defer os.Remove(tempPath)
		m := manifest.New(tempPath)
		return m, m.Read()
	} else if err != nil {
		return nil, err
	}
	m := manifest.New(from)
	return m, m.Read()
}

func fetch(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("CRD request failed: %s", resp.Status)
	}
	file, err := os.CreateTemp("", "")
	if err != nil {
		return "", err
	}
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", err
	}
	return file.Name(), nil
}

func parseUserInputTags(name, kind string, spec map[string]interface{}) (map[string]interface{}, error) {
	filledSpec := make(map[string]interface{}, len(spec))
	for key, value := range spec {
		switch v := value.(type) {
		case string:
			if v != triggermesh.UserInputTag {
				filledSpec[key] = v
				continue
			}
			fmt.Printf("%s/%s: ", name, key)
			input, err := readStdin()
			if err != nil {
				return nil, err
			}
			if kind == "Secret" {
				input = base64.StdEncoding.EncodeToString([]byte(input))
			}
			filledSpec[key] = input
		case map[string]interface{}:
			filled, err := parseUserInputTags(name, kind, v)
			if err != nil {
				return nil, err
			}
			filledSpec[key] = filled
		}
	}
	return filledSpec, nil
}

func readStdin() (string, error) {
	var line string
	scn := bufio.NewScanner(os.Stdin)
	for scn.Scan() {
		line = scn.Text()
		break
	}
	return line, scn.Err()
}
