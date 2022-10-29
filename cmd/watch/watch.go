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

package watch

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/triggermesh/tmctl/pkg/wiretap"
)

type WatchOptions struct {
	ConfigDir  string
	EventTypes string
	Source     string
}

func NewCmd() *cobra.Command {
	o := WatchOptions{}
	watchCmd := &cobra.Command{
		Use:   "watch [broker]",
		Short: "Watch events flowing through the broker",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			return []string{}, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			o.ConfigDir = path.Dir(viper.ConfigFileUsed())
			if len(args) == 1 {
				return o.watch(args[0])
			}
			return o.watch(viper.GetString("context"))
		},
	}

	watchCmd.Flags().StringVarP(&o.EventTypes, "eventTypes", "e", "", "Filter events based on type attribute")

	return watchCmd
}

func (o *WatchOptions) watch(broker string) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	ctx := context.Background()

	w := wiretap.New(broker, o.ConfigDir)
	defer func() {
		if err := w.Cleanup(ctx); err != nil {
			log.Printf("Cleanup: %v", err)
		}
	}()
	logs, err := w.CreateAdapter(ctx)
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}
	if err := w.CreateTrigger(strings.Split(o.EventTypes, ",")); err != nil {
		return fmt.Errorf("create trigger: %w", err)
	}
	log.Println("Watching...")
	go listenLogs(logs, c)
	<-c
	log.Println("Exiting")
	return nil
}

func listenLogs(output io.ReadCloser, done chan os.Signal) {
	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		select {
		case <-done:
			output.Close()
			return
		default:
			log := scanner.Bytes()
			if len(log) > 8 {
				log = log[8:]
			}
			fmt.Println(string(log))
		}
	}
}
