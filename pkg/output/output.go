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

package output

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/triggermesh/tmcli/pkg/docker"
	"github.com/triggermesh/tmcli/pkg/triggermesh"
)

const (
	delimeter = "---------------"

	successColorCode = "\033[92m"
	defaultColorCode = "\033[39m"
	offlineColorCode = "\u001b[31m"
)

var w *tabwriter.Writer

func init() {
	w = tabwriter.NewWriter(os.Stdout, 10, 5, 5, ' ', 0)
}

func PrintStatus(kind string, object triggermesh.Component, sourceName string, eventTypesFilter []string) {
	var result string
	result = fmt.Sprintf("%s\nCreated object name:\t%s", delimeter, object.GetName())

	switch kind {
	case "broker":
		result = fmt.Sprintf("%s\nCurrent context is set to %q", result, object.GetName())
		result = fmt.Sprintf("%s\nTo change the context use \"tmcli config set context <context name>\"", result)
		result = fmt.Sprintf("%s\n\nNext steps:", result)
		result = fmt.Sprintf("%s\n\ttmcli create source\t - create source that will produce events", result)
	case "producer":
		et, _ := object.(triggermesh.Producer).GetEventTypes()
		if len(et) != 0 {
			result = fmt.Sprintf("%s\nComponent produces:\t%s", result, strings.Join(et, ", "))
		}
		result = fmt.Sprintf("%s\n\nNext steps:", result)
		result = fmt.Sprintf("%s\n\ttmcli create target <kind> --source %s [--eventTypes <types>]\t - create target that will consume events from this source", result, object.GetName())
		result = fmt.Sprintf("%s\n\ttmcli watch\t\t\t\t\t\t\t\t\t - show events flowing through the broker in the real time", result)
	case "consumer":
		et, _ := object.(triggermesh.Consumer).ConsumedEventTypes()
		if len(et) != 0 {
			result = fmt.Sprintf("%s\nComponent consumes:\t%s", result, strings.Join(et, ", "))
		}
		srcMsg := strings.Join(eventTypesFilter, ", ")
		if sourceName != "" {
			srcMsg = fmt.Sprintf("%s(%s)", sourceName, srcMsg)
		}
		result = fmt.Sprintf("%s\nSubscribed to:\t\t%s", result, srcMsg)
		result = fmt.Sprintf("%s\n\nNext steps:", result)
		result = fmt.Sprintf("%s\n\ttmcli watch\t - show events flowing through the broker in the real time", result)
		result = fmt.Sprintf("%s\n\ttmcli dump\t - dump Kubernetes manifest", result)
	}

	fmt.Printf("%s%s\n%s\n%s", successColorCode, result, delimeter, defaultColorCode)
}

// func Draw() {}
// func Dump() {}

func status(container *docker.Container) string {
	status := fmt.Sprintf("%soffline%s", offlineColorCode, defaultColorCode)
	if container != nil {
		status = fmt.Sprintf("%sonline(http://localhost:%s)%s", successColorCode, container.HostPort(), defaultColorCode)
	}
	return status
}

func DescribeBroker(brokers []triggermesh.Component, containers []*docker.Container) {
	defer w.Flush()
	fmt.Fprintln(w, "Broker\tStatus")
	for i, broker := range brokers {
		fmt.Fprintf(w, "%s\t%s\n", broker.GetName(), status(containers[i]))
	}
	fmt.Fprintln(w)
}

func DescribeSource(sources []triggermesh.Component, containers []*docker.Container) {
	if len(sources) == 0 {
		return
	}
	defer w.Flush()
	fmt.Fprintln(w, "Source\tKind\tEventTypes\tStatus")
	for i, source := range sources {
		et, err := source.(triggermesh.Producer).GetEventTypes()
		if err != nil {
			et = []string{"-"}
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", source.GetName(), source.GetKind(), strings.Join(et, ","), status(containers[i]))
	}
	fmt.Fprintln(w)
}

func DescribeTransformation(transformations []triggermesh.Component, containers []*docker.Container) {
	if len(transformations) == 0 {
		return
	}
	defer w.Flush()
	fmt.Fprintln(w, "Transformation\tEventTypes\tStatus")
	for i, transformation := range transformations {
		et, err := transformation.(triggermesh.Producer).GetEventTypes()
		if err != nil {
			et = []string{"-"}
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", transformation.GetName(), strings.Join(et, ","), status(containers[i]))
	}
	fmt.Fprintln(w)
}

func DescribeTarget(targets []triggermesh.Component, containers []*docker.Container) {
	if len(targets) == 0 {
		return
	}
	defer w.Flush()
	fmt.Fprintln(w, "Target\tKind\tStatus")
	for i, target := range targets {
		fmt.Fprintf(w, "%s\t%s\t%s\n", target.GetName(), target.GetKind(), status(containers[i]))
	}
	fmt.Fprintln(w)
}
