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

	"github.com/triggermesh/tmcli/pkg/triggermesh"
)

const delimeter = "---------------"

func ComponentStatus(kind string, object triggermesh.Component, sourceName string, eventTypesFilter []string) string {
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
	return fmt.Sprintf("%s\n%s", result, delimeter)
}

// func Draw() {}
// func Dump() {}
