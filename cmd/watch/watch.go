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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/triggermesh/tmctl/pkg/config"
	"github.com/triggermesh/tmctl/pkg/log"
	"github.com/triggermesh/tmctl/pkg/wiretap"
)

type CliOptions struct {
	Config *config.Config

	EventTypes string
	Source     string
}

type brokerLog struct {
	Level  string `json:"level"`
	Logger string `json:"logger"`
	Msg    string `json:"msg"`
	Name   string `json:"name"`
}

func NewCmd(config *config.Config) *cobra.Command {
	o := &CliOptions{Config: config}
	watchCmd := &cobra.Command{
		Use:     "watch [broker]",
		Short:   "Watch events flowing through the broker",
		Example: "tmctl watch",
		Args:    cobra.RangeArgs(0, 1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			return []string{}, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				o.Config.Context = args[0]
			}
			return o.watch()
		},
	}
	watchCmd.Flags().StringVarP(&o.EventTypes, "eventTypes", "e", "", "Filter events based on type attribute")
	return watchCmd
}

func (o *CliOptions) watch() error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	ctx := context.Background()

	w, err := wiretap.New(o.Config.Context, o.Config.ConfigHome)
	if err != nil {
		return fmt.Errorf("wiretap: %w", err)
	}
	defer func() {
		if err := w.Cleanup(ctx); err != nil {
			log.Printf("Cleanup: %v", err)
		}
	}()
	log.Println("Connecting to broker")
	eventDisplayLogs, err := w.CreateAdapter(ctx)
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}
	if err := w.CreateTrigger(strings.Split(o.EventTypes, ",")); err != nil {
		return fmt.Errorf("create trigger: %w", err)
	}
	brokerLogs, err := w.BrokerLogs(ctx, o.Config.Triggermesh.Broker)
	if err != nil {
		return fmt.Errorf("broker logs: %w", err)
	}
	log.Println("Watching...")
	go listenBroker(brokerLogs, c)
	go listenEvents(eventDisplayLogs, c)
	<-c
	log.Println("Cleaning up")
	return nil
}

func listenEvents(output io.ReadCloser, done chan os.Signal) {
	readLogs(output, done, func(data []byte) {
		fmt.Println(string(data))
	})
}

func listenBroker(output io.ReadCloser, done chan os.Signal) {
	readLogs(output, done, func(data []byte) {
		var logItem brokerLog
		if err := json.Unmarshal(data, &logItem); err != nil {
			return
		}
		if logItem.Level == "error" {
			fmt.Printf("❗ error: %s", logItem)
			return
		}
		if logItem.Logger == "subs" {
			fmt.Printf("🔧 configuration: %s: %s\n", logItem.Msg, logItem.Name)
		}
	})
}

func readLogs(output io.ReadCloser, done chan os.Signal, handler func([]byte)) {
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
			handler(log)
		}
	}
}
