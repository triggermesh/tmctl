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
	"strings"

	"github.com/triggermesh/tmctl/pkg/triggermesh"
)

const (
	delimeter = "---------------"

	successColorCode = "\033[92m"
	defaultColorCode = "\033[39m"
)

func PrintStatus(kind string, object triggermesh.Component, eventSourcesFilter, eventTypesFilter []string) {
	var result string
	result = fmt.Sprintf("%s\nCreated object name:\t%s", delimeter, object.GetName())

	switch kind {
	case "broker":
		result = fmt.Sprintf("%s\nCurrent broker is set to %q", result, object.GetName())
		result = fmt.Sprintf("%s\nTo change the current broker use \"tmctl brokers --set <broker name>\"", result)
		result = fmt.Sprintf("%s%s\n%s", successColorCode, result, defaultColorCode)
		// result = fmt.Sprintf("%s\nNext steps:", result)
		// result = fmt.Sprintf("%s\n\ttmctl create source\t - create source that will produce events", result)
	case "producer":
		et, _ := object.(triggermesh.Producer).GetEventTypes()
		if len(et) != 0 {
			result = fmt.Sprintf("%s\nComponent produces:\t%s", result, strings.Join(et, ", "))
		}
		result = fmt.Sprintf("%s%s\n%s", successColorCode, result, defaultColorCode)
		// result = fmt.Sprintf("%s\nNext steps:", result)
		// result = fmt.Sprintf("%s\n\ttmctl create target <kind> --source %s [--event-types <types>]\t - create target that will consume events from this source", result, object.GetName())
		// result = fmt.Sprintf("%s\n\ttmctl watch\t\t\t\t\t\t\t\t\t - show events flowing through the broker in the real time", result)
	case "consumer":
		et, _ := object.(triggermesh.Consumer).ConsumedEventTypes()
		if len(et) != 0 {
			result = fmt.Sprintf("%s\nComponent consumes:\t%s", result, strings.Join(et, ", "))
		}
		filter := strings.Join(eventTypesFilter, ", ")
		if len(eventSourcesFilter) != 0 {
			filter = fmt.Sprintf("%s(%s)", strings.Join(eventSourcesFilter, ", "), filter)
		}
		if filter != "" {
			result = fmt.Sprintf("%s\nSubscribed to:\t\t%s", result, filter)
		}
		result = fmt.Sprintf("%s%s\n%s", successColorCode, result, defaultColorCode)
		// result = fmt.Sprintf("%s\nNext steps:", result)
		// result = fmt.Sprintf("%s\n\ttmctl create transformation --target %s\t - create event transformation component", result, object.GetName())
		// result = fmt.Sprintf("%s\n\ttmctl create trigger --target %s\t - create trigger to send events from source to target", result, object.GetName())
		// result = fmt.Sprintf("%s\n\ttmctl dump\t - dump Kubernetes manifest", result)
	}
	fmt.Print(result)
}

// func Draw() {}
// func Dump() {}
