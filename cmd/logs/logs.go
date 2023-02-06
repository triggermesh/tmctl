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

package logs

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/triggermesh/tmctl/pkg/completion"
	"github.com/triggermesh/tmctl/pkg/config"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

const defaultColorCode = "\033[0m"

var colors = []string{
	"\033[31m",   // red
	"\033[32m",   // green
	"\033[33m",   // yellow
	"\033[34m",   // blue
	"\033[35m",   // magent
	"\033[36m",   // cyan
	"\033[31;1m", // bright red
	"\033[32;1m", // bright green
	"\033[33;1m", // bright yellow
	"\033[34;1m", // bright blue
	"\033[35;1m", // bright magent
	"\033[36;1m", // bright cyan
}

var defaultLogPeriod = 24 * time.Hour

type CliOptions struct {
	Config   *config.Config
	Manifest *manifest.Manifest
	CRD      map[string]crd.CRD
}

func NewCmd(config *config.Config, manifest *manifest.Manifest, crd map[string]crd.CRD) *cobra.Command {
	o := &CliOptions{
		CRD:      crd,
		Config:   config,
		Manifest: manifest,
	}
	var follow bool
	logsCmd := &cobra.Command{
		Use:     "logs [name]",
		Short:   "Display components logs",
		Example: "tmctl logs",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			return completion.ListAll(o.Manifest), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cobra.CheckErr(o.Manifest.Read())
			return o.logs(args, follow)
		},
	}
	logsCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow logs output")
	return logsCmd
}

func (o *CliOptions) logs(filter []string, follow bool) error {
	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, os.Interrupt, syscall.SIGTERM)
	defer close(cancel)

	ctx := context.Background()

	colorIndex := 0
	for _, object := range o.Manifest.Objects {
		component, err := components.GetObject(object.Metadata.Name, o.Config, o.Manifest, o.CRD)
		if err != nil {
			return fmt.Errorf("creating component interface: %w", err)
		}
		if component == nil {
			continue
		}
		if len(filter) != 0 {
			listed := false
			for _, name := range filter {
				if component.GetName() == name {
					listed = true
					break
				}
			}
			if !listed {
				continue
			}
		}
		container, ok := component.(triggermesh.Runnable)
		if !ok {
			continue
		}
		since := time.Now()
		if !follow {
			since = since.Add(-defaultLogPeriod * time.Hour)
		}
		logs, err := container.Logs(ctx, since, follow)
		if err != nil {
			return fmt.Errorf("%q logs unavailable: %w", component.GetName(), err)
		}
		defer logs.Close()
		colorCode := func() string {
			if len(filter) == 1 {
				return defaultColorCode
			}
			if colorIndex >= len(colors) {
				colorIndex -= len(colors)
			}
			return colors[colorIndex]
		}()
		colorIndex++
		if follow {
			log.Printf("%sListening %s%s", colorCode, component.GetName(), defaultColorCode)
			go readLogs(logs, cancel, colorCode)
		} else {
			fmt.Printf("---------------\n%s\n---------------\n", component.GetName())
			readLogs(logs, cancel, defaultColorCode)
		}
	}
	if follow {
		<-cancel
	}
	return nil
}

func readLogs(logs io.ReadCloser, calncel chan os.Signal, colorCode string) {
	defer logs.Close()
	scanner := bufio.NewScanner(logs)
	for scanner.Scan() {
		select {
		case <-calncel:
			return
		default:
			log := scanner.Bytes()
			if len(log) > 8 {
				log = log[8:]
			}
			fmt.Printf("%s%s%s\n", colorCode, string(log), defaultColorCode)
		}
	}
}
