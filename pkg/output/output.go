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

	"github.com/triggermesh/tmctl/pkg/docker"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
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

func PrintStatus(kind string, object triggermesh.Component, eventSourcesFilter, eventTypesFilter []string) {
	var result string
	result = fmt.Sprintf("%s\nCreated object name:\t%s", delimeter, object.GetName())

	switch kind {
	case "broker":
		result = fmt.Sprintf("%s\nCurrent broker is set to %q", result, object.GetName())
		result = fmt.Sprintf("%s\nTo change the current broker use \"tmctl brokers --set <broker name>\"", result)
		result = fmt.Sprintf("%s%s\n%s%s", successColorCode, result, delimeter, defaultColorCode)
		result = fmt.Sprintf("%s\nNext steps:", result)
		result = fmt.Sprintf("%s\n\ttmctl create source\t - create source that will produce events", result)
	case "producer":
		et, _ := object.(triggermesh.Producer).GetEventTypes()
		if len(et) != 0 {
			result = fmt.Sprintf("%s\nComponent produces:\t%s", result, strings.Join(et, ", "))
		}
		result = fmt.Sprintf("%s%s\n%s%s", successColorCode, result, delimeter, defaultColorCode)
		result = fmt.Sprintf("%s\nNext steps:", result)
		result = fmt.Sprintf("%s\n\ttmctl create target <kind> --source %s [--eventTypes <types>]\t - create target that will consume events from this source", result, object.GetName())
		result = fmt.Sprintf("%s\n\ttmctl watch\t\t\t\t\t\t\t\t\t - show events flowing through the broker in the real time", result)
	case "consumer":
		et, _ := object.(triggermesh.Consumer).ConsumedEventTypes()
		if len(et) != 0 {
			result = fmt.Sprintf("%s\nComponent consumes:\t%s", result, strings.Join(et, ", "))
		}
		filter := strings.Join(append(eventTypesFilter, eventSourcesFilter...), ", ")
		if filter != "" {
			result = fmt.Sprintf("%s\nSubscribed to:\t\t%s", result, filter)
		}
		result = fmt.Sprintf("%s%s\n%s%s", successColorCode, result, delimeter, defaultColorCode)
		result = fmt.Sprintf("%s\nNext steps:", result)
		result = fmt.Sprintf("%s\n\ttmctl watch\t - show events flowing through the broker in the real time", result)
		result = fmt.Sprintf("%s\n\ttmctl dump\t - dump Kubernetes manifest", result)
	}
	fmt.Println(result)
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
	fmt.Fprintln(w, "Source\tKind\tEventSource\tEventTypes\tStatus")
	for i, source := range sources {
		et, _ := source.(triggermesh.Producer).GetEventTypes()
		if len(et) == 0 {
			et = []string{"-"}
		}
		es, _ := source.(triggermesh.Producer).GetEventSource()
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", source.GetName(), source.GetKind(), es, strings.Join(et, ","), status(containers[i]))
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

func DescribeTrigger(triggers []triggermesh.Component) {
	if len(triggers) == 0 {
		return
	}
	defer w.Flush()
	fmt.Fprintln(w, "Trigger\tTarget\tFilter")
	for _, trigger := range triggers {
		var filters []string
		for _, filter := range trigger.(*tmbroker.Trigger).GetFilters() {
			filters = append(filters, triggerFilterToString(filter))
		}
		if len(filters) == 0 {
			filters = []string{"*"}
		}
		fmt.Fprintf(w, "%s\t%v\t%v\n", trigger.GetName(), trigger.(*tmbroker.Trigger).GetTarget().Component, strings.Join(filters, ", "))
	}
	fmt.Fprintln(w)
}

func triggerFilterToString(filter tmbroker.Filter) string {
	var result []string
	for k, v := range filter.Exact {
		result = append(result, fmt.Sprintf("%s is %s", k, v))
	}
	return strings.Join(result, ", ")
}
