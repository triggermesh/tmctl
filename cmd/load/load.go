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

package load

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/triggermesh/tmctl/cmd/describe"
	"github.com/triggermesh/tmctl/pkg/config"
	"github.com/triggermesh/tmctl/pkg/log"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/secret"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

const (
	successColorCode = "\033[92m"
	defaultColorCode = "\033[39m"
)

type CliOptions struct {
	Config *config.Config
	CRD    map[string]crd.CRD
}

func NewCmd(config *config.Config, crd map[string]crd.CRD) *cobra.Command {
	o := &CliOptions{
		CRD:    crd,
		Config: config,
	}
	var from string
	importCmd := &cobra.Command{
		Use:     "import -f <path/to/manifest.yaml>/<manifest URL>",
		Short:   "Import TriggerMesh manifest",
		Example: "tmctl import -f manifest.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.importManifest(from)
		},
	}
	importCmd.Flags().StringVarP(&from, "from", "f", "", "Import manifest from")
	importCmd.MarkFlagRequired("f")
	return importCmd
}

func (o *CliOptions) importManifest(from string) error {
	m, err := getManifest(from)
	if err != nil {
		return fmt.Errorf("manifest %q: %w", from, err)
	}

	// create broker first
	brokerName := ""
	for _, object := range m.Objects {
		if object.Kind != tmbroker.BrokerKind {
			continue
		}
		if _, err := tmbroker.CreateBrokerConfig(o.Config.ConfigHome, object.Metadata.Name); err != nil {
			return fmt.Errorf("creating broker object: %w", err)
		}
		b, err := tmbroker.New(object.Metadata.Name, o.Config.Triggermesh.Broker)
		if err != nil {
			return fmt.Errorf("creating broker object: %w", err)
		}
		brokerName = b.GetName()
		break
	}

	m.Path = filepath.Join(o.Config.ConfigHome, brokerName, triggermesh.ManifestFile)
	if err := m.Write(); err != nil {
		return err
	}

	// then write broker config
	for _, object := range m.Objects {
		if object.Kind != tmbroker.TriggerKind {
			continue
		}
		trigger, err := components.GetObject(object.Metadata.Name, o.Config, m, o.CRD)
		if err != nil {
			return err
		}
		if err := trigger.(*tmbroker.Trigger).WriteLocalConfig(); err != nil {
			return err
		}
	}

	// update redacted secrets
	for _, object := range m.Objects {
		if object.Kind != "Secret" {
			continue
		}
		component, err := components.GetObject(object.Metadata.Name, o.Config, m, o.CRD)
		if err != nil {
			return err
		}
		data := component.(*secret.Secret).GetSpec()
		filledData := make(map[string]string, len(data))
		for k, v := range data {
			filledData[k] = v.(string)
			if v.(string) == "<redacted>" {
				fmt.Printf("%s/%s (plain string): ", component.GetName(), k)
				input, err := readToBase64()
				if err != nil {
					return err
				}
				filledData[k] = input
			}
		}
		component = secret.New(component.GetName(), brokerName, filledData)
		if _, err := m.Add(component); err != nil {
			return err
		}
	}

	descr := &describe.CliOptions{
		Config:   o.Config,
		Manifest: m,
		CRD:      o.CRD,
	}
	_ = descr.Describe()

	config.Set("context", brokerName)
	log.Printf("Context switched")
	log.Printf("%sBroker %q imported successfully%s", successColorCode, brokerName, defaultColorCode)
	return nil
}

func readToBase64() (string, error) {
	var line string
	scn := bufio.NewScanner(os.Stdin)
	for scn.Scan() {
		line = scn.Text()
		break
	}
	return base64.StdEncoding.EncodeToString([]byte(line)), scn.Err()
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
	file, err := ioutil.TempFile("", "")
	if err != nil {
		return "", err
	}
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", err
	}
	return file.Name(), nil
}
