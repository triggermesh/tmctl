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

package dump

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/mitchellh/mapstructure"
	"github.com/robertkrimen/otto"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/triggermesh/tmctl/pkg/manifest"
)

const manifestFile = "manifest.yaml"

type DumpOptions struct {
	ConfigDir string
	Format    string
}

type DockerService struct {
	Name               string            `json:"name",yaml:"name"`
	Image              string            `json:"image", yaml:"image"`
	Environment        []interface{}     `json:"environment", yaml:"environment"`
	Ports              []interface{}     `json:"ports", yaml:"ports"`
	Volumes            []interface{}     `json:"volumes", yaml:"volumes"`
	VolumeMounts       []interface{}     `json:"volumeMounts", yaml:"volumeMounts"`
	Secrets            []interface{}     `json:"secrets", yaml:"secrets"`
	SecretVolumeMounts []interface{}     `json:"secretVolumeMounts", yaml:"secretVolumeMounts"`
	ConfigVolumeMounts []interface{}     `json:"configVolumeMounts", yaml:"configVolumeMounts"`
	ConfigVolumes      []interface{}     `json:"configVolumes", yaml:"configVolumes"`
	Configmaps         map[string]string `json:"configmaps", yaml:"configmaps"`
	Labels             map[string]string `json:"labels", yaml:"labels"`
}

type Auth struct {
	AccessKeyID     string `json:"accessKeyID",yaml:"accessKeyID"`
	SecretAccessKey string `json:"secretAccessKey",yaml:"secretAccessKey"`
}

type AWSAuth struct {
	Auth Auth `json:"auth",yaml:"auth"`
}

func NewCmd() *cobra.Command {
	o := &DumpOptions{}
	dumpCmd := &cobra.Command{
		Use:   "dump [broker]",
		Short: "Generate Kubernetes manifest",
		RunE: func(cmd *cobra.Command, args []string) error {
			o.ConfigDir = path.Dir(viper.ConfigFileUsed())
			if len(args) == 1 {
				return o.dump(args[0])
			}
			return o.dump(viper.GetString("context"))
		},
	}
	dumpCmd.Flags().StringVarP(&o.Format, "output", "o", "yaml", "Output format")
	return dumpCmd
}

func (o *DumpOptions) dump(broker string) error {
	manifest := manifest.New(path.Join(o.ConfigDir, broker, manifestFile))
	if err := manifest.Read(); err != nil {
		return err
	}
	switch o.Format {
	case "json":
		for _, v := range manifest.Objects {
			jsn, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(jsn))
		}
	case "yaml":
		for _, v := range manifest.Objects {
			yml, err := yaml.Marshal(v)
			if err != nil {
				return err
			}
			fmt.Println("---")
			fmt.Println(string(yml))
		}
	case "docker-compose":

		/// inspect each of the runing containers and break them dwn
		//  C:\Users\jnlas\go\src\github.com\triggermesh\tmcli\pkg\triggermesh\interfaces.go
		// C:\Users\jnlas\go\src\github.com\triggermesh\tmcli\pkg\triggermesh\components

		fmt.Println("Services:")
		// var dockerServices []DockerService
		// var jsn []byte
		// var err error
		vm := otto.New()
		for _, v := range manifest.Objects {
			// open a javascript interpreter
			// run the javascript
			// get the result
			// display the kubernetes object as a docker-compose service
			// {"APIVersion":"eventing.triggermesh.io/v1alpha1","Data":{},"Kind":"Broker","Metadata":{"Labels":{"triggermesh.io/context":"bob"},"Name":"bob","Namespace":"","OwnerReferences":[]},"Spec":{"storage":"inmemory"},"Type":""}
			vm.Set("obj", v)
			vm.Run(`
			console.log("  " + obj.Metadata.Name)
			console.log("    image: " + obj.Spec.Image)
			console.log("    environment:")
			for (var key in obj.Spec) {
			// check for nested objects
			if (typeof obj.Spec[key] === 'object') {
				for (var subkey in obj.Spec[key]) {
					if (typeof obj.Spec[key][subkey] === 'object') {
						for (var subsubkey in obj.Spec[key][subkey]) {
							console.log("      " + subsubkey + ": " + obj.Spec[key][subkey][subsubkey])
						}
					} else {
						console.log("      " + subkey + ": " + obj.Spec[key][subkey])
					}
				}
			} else {
			console.log("      " + key + ": " + obj.Spec[key])
			}
			}
			console.log("    ports:")
			console.log("      - 8080 ")
			console.log("    volumes:")
			for (var key in obj.Spec.Volumes) {
				console.log("      - " + obj.Spec.Volumes[key])
			}
			console.log("    volumeMounts:")
			for (var key in obj.Spec.VolumeMounts) {
				console.log("      - " + obj.Spec.VolumeMounts[key])
			}

			`)
		}

		// 	// append the map to the array
		// 	jsn = append(jsn, json...)
		// }
		// var dockerServices []DockerService
		// err = json.Unmarshal(jsn, &dockerServices)
		// if err != nil {
		// 	return err
		// }
		// open javascript interpreter

		// run the javascript to parse the json
		// and display the result in docker-compose format
		// vm.Set("dockerServices", manifest.Objects)
		// vm.Run(`
		// for (var i = 0; i < dockerServices.length; i++) {
		// 	console.log(dockerServices[i].name + ":");
		// 	console.log("  image: " + dockerServices[i].image);
		// 	console.log("  environment:");
		// }`)

		// for (var i = 0; i < dockerServices.length; i++) {
		// 	console.log(dockerServices[i].name + ":");
		// 	console.log("  image: " + dockerServices[i].image);
		// console.log("  environment:");
		// for (var j = 0; j < dockerServices[i].environment.length; j++) {
		// 	console.log("    - " + dockerServices[i].environment[j].name + "=" + dockerServices[i].environment[j].value);
		// }
		// console.log("  ports:");
		// for (var j = 0; j < dockerServices[i].ports.length; j++) {
		// 	console.log("    - " + dockerServices[i].ports[j].containerPort + ":" + dockerServices[i].ports[j].containerPort);
		// }
		// console.log("  volumes:");
		// for (var j = 0; j < dockerServices[i].volumes.length; j++) {
		// 	console.log("    - " + dockerServices[i].volumes[j].name + ":" + dockerServices[i].volumes[j].mountPath);
		// }
		// console.log("  volumeMounts:");
		// for (var j = 0; j < dockerServices[i].volumeMounts.length; j++) {
		// 	console.log("    - " + dockerServices[i].volumeMounts[j].name + ":" + dockerServices[i].volumeMounts[j].mountPath);
		// }
		// console.log("  secrets:");
		// for (var j = 0; j < dockerServices[i].secrets.length; j++) {
		// 	console.log("    - " + dockerServices[i].secrets[j].secretName + ":" + dockerServices[i].secrets[j].mountPath);
		// }
		// console.log("  secretVolumeMounts:");
		// for (var j = 0; j < dockerServices[i].secretVolumeMounts.length; j++) {
		// 	console.log("    - " + dockerServices[i].secretVolumeMounts[j].secretName + ":" + dockerServices[i].secretVolumeMounts[j].mountPath);
		// }
		// console.log("  configVolumeMounts:");
		// for (var j = 0; j < dockerServices[i].configVolumeMounts.length; j++) {
		// 	console.log("    - " + dockerServices[i].configVolumeMounts[j].configMapName

		// for _, v := range dockerServices {
		// 	fmt.Println(v)
		// }

	default:
		return fmt.Errorf("format %q is not supported", o.Format)
	}

	return nil
}

func printEnv(m map[string]interface{}, prefix string) {
	for k, v := range m {
		if m, ok := v.(map[string]interface{}); ok {
			printEnv(m, k)
		} else {
			fmt.Println("      " + k + ": " + fmt.Sprintf("%+v", v))
		}
	}
}

func env(ds map[string]interface{}) {
	for k, v := range ds {
		// Transformations will have a super ugly spec in docker compose
		// look for an "auth" key and print the accessKeyID and secretAccessKey
		// still broken :/
		if k == "auth" {
			awsAuth := AWSAuth{}
			if err := mapstructure.Decode(v, &awsAuth); err != nil {
				panic(err)
			}
			fmt.Println("      " + k + ": " + fmt.Sprintf("%v", awsAuth.Auth.AccessKeyID))
			fmt.Println("      " + k + ": " + fmt.Sprintf("%v", awsAuth.Auth.SecretAccessKey))
		} else {
			if m, ok := v.(map[string]interface{}); ok {

				fmt.Println("      " + k + ": " + fmt.Sprintf("%v", m))

			} else {
				fmt.Println("      " + k + ": " + fmt.Sprintf("%v", v))
			}
		}
	}
}
