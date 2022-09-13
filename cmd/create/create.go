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

package create

import (
	"fmt"

	"github.com/spf13/cobra"
)

const manifestFile = "manifest.yaml"

type CreateOptions struct {
	ConfigBase string
	Context    string
	Version    string
	CRD        string
}

func NewCmd() *cobra.Command {
	o := &CreateOptions{}
	createCmd := &cobra.Command{
		Use:   "create <resource>",
		Short: "create TriggerMesh objects",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}
	createCmd.AddCommand(o.NewBrokerCmd())
	createCmd.AddCommand(o.NewSourceCmd())
	createCmd.AddCommand(o.NewTargetCmd())
	createCmd.AddCommand(o.NewTriggerCmd())

	createCmd.Flags().StringVarP(&o.Context, "broker", "b", "", "Connect components to this broker")

	return createCmd
}

func parse(args []string) (string, []string, error) {
	if l := len(args); l < 1 {
		return "", []string{}, fmt.Errorf("expected at least 1 arguments, got %d", l)
	}
	return args[0], args[1:], nil
}

// Function init:
//    if image, err = function.ImageName(k8sObject); err != nil {
//            return "", fmt.Errorf("cannot parse function image: %w", err)
//    }
//    image = fmt.Sprintf("%s:%s", image, version)
//    file, err := createSharedFile(function.Code(k8sObject))
//    if err != nil {
//            return "", fmt.Errorf("writing function: %w", err)
//    }
//    bind := fmt.Sprintf("%s:/opt/source.%s", file.Name(), function.FileExtension(k8sObject))
//    hostOptions = append(hostOptions, d.WithVolumeBind(bind))
//    containerOptions = append(containerOptions, d.WithEntrypoint("/opt/aws-custom-runtime"))
