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
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/triggermesh/tmcli/pkg/triggermesh/wiretap"
)

type WatchOptions struct {
	ConfigDir string
	Context   string
}

func NewCmd() *cobra.Command {
	o := WatchOptions{}
	watchCmd := &cobra.Command{
		Use:   "watch <broker>",
		Short: "Watch events flowing through the broker",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}
			o.ConfigDir = c
			o.Context = viper.GetString("context")
			return o.watch()
		},
	}
	return watchCmd
}

func (o *WatchOptions) watch() error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	ctx := context.Background()

	w := wiretap.New(o.Context, o.ConfigDir)
	defer func() {
		if err := w.Cleanup(ctx); err != nil {
			log.Printf("Cleanup: %v", err)
		}
	}()
	logs, err := w.CreateAdapter(ctx)
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}
	if err := w.CreateTrigger(); err != nil {
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
			fmt.Println(string(scanner.Bytes()[8:]))
		}
	}
}
