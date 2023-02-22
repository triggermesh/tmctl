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

package ce

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	targetsv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
)

func targets(object unstructured.Unstructured) (EventAttributes, error) {
	switch object.GetKind() {
	case "AWSComprehendTarget":
		var o *targetsv1alpha1.AWSComprehendTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "AWSDynamoDBTarget":
		var o *targetsv1alpha1.AWSDynamoDBTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "AWSEventBridgeTarget":
		var o *targetsv1alpha1.AWSEventBridgeTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
			AcceptedEventTypes:  o.AcceptedEventTypes(),
		}, nil
	case "AWSKinesisTarget":
		return EventAttributes{}, nil
	case "AWSLambdaTarget":
		return EventAttributes{}, nil
	case "AWSS3Target":
		var o *targetsv1alpha1.AWSS3Target
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
			AcceptedEventTypes:  o.AcceptedEventTypes(),
		}, nil
	case "AWSSNSTarget":
		return EventAttributes{}, nil
	case "AWSSQSTarget":
		return EventAttributes{}, nil
	case "AzureEventHubsTarget":
		var o *targetsv1alpha1.AzureEventHubsTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
			AcceptedEventTypes:  o.AcceptedEventTypes(),
		}, nil
	case "CloudEventsTarget":
		return EventAttributes{}, nil
	case "ConfluentTarget":
		return EventAttributes{}, nil
	case "DatadogTarget":
		var o *targetsv1alpha1.DatadogTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
			AcceptedEventTypes:  o.AcceptedEventTypes(),
		}, nil
	case "ElasticsearchTarget":
		var o *targetsv1alpha1.ElasticsearchTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
			AcceptedEventTypes:  o.AcceptedEventTypes(),
		}, nil
	case "GoogleCloudFirestoreTarget":
		var o *targetsv1alpha1.GoogleCloudFirestoreTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
			AcceptedEventTypes:  o.AcceptedEventTypes(),
		}, nil
	case "GoogleCloudStorageTarget":
		var o *targetsv1alpha1.GoogleCloudStorageTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
			AcceptedEventTypes:  o.AcceptedEventTypes(),
		}, nil
	case "GoogleCloudWorkflowsTarget":
		var o *targetsv1alpha1.GoogleCloudWorkflowsTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
			AcceptedEventTypes:  o.AcceptedEventTypes(),
		}, nil
	case "GoogleSheetTarget":
		var o *targetsv1alpha1.GoogleSheetTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
			AcceptedEventTypes:  o.AcceptedEventTypes(),
		}, nil
	case "HTTPTarget":
		return EventAttributes{}, nil
	case "IBMMQTarget":
		var o *targetsv1alpha1.IBMMQTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
			AcceptedEventTypes:  o.AcceptedEventTypes(),
		}, nil
	case "JiraTarget":
		var o *targetsv1alpha1.JiraTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
			AcceptedEventTypes:  o.AcceptedEventTypes(),
		}, nil
	case "KafkaTarget":
		return EventAttributes{}, nil
	case "LogzMetricsTarget":
		var o *targetsv1alpha1.LogzMetricsTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			AcceptedEventTypes: o.AcceptedEventTypes(),
		}, nil
	case "LogzTarget":
		var o *targetsv1alpha1.LogzTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "MongoDBTarget":
		var o *targetsv1alpha1.MongoDBTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
			AcceptedEventTypes:  o.AcceptedEventTypes(),
		}, nil
	case "OracleTarget":
		return EventAttributes{}, nil
	case "SalesforceTarget":
		var o *targetsv1alpha1.SalesforceTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
			AcceptedEventTypes:  o.AcceptedEventTypes(),
		}, nil
	case "SendGridTarget":
		var o *targetsv1alpha1.SendGridTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
			AcceptedEventTypes:  o.AcceptedEventTypes(),
		}, nil
	case "SlackTarget":
		var o *targetsv1alpha1.SlackTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
			AcceptedEventTypes:  o.AcceptedEventTypes(),
		}, nil
	case "SolaceTarget":
		return EventAttributes{}, nil
	case "SplunkTarget":
		return EventAttributes{}, nil
	case "TwilioTarget":
		var o *targetsv1alpha1.TwilioTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
			AcceptedEventTypes:  o.AcceptedEventTypes(),
		}, nil
	case "ZendeskTarget":
		var o *targetsv1alpha1.ZendeskTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
			AcceptedEventTypes:  o.AcceptedEventTypes(),
		}, nil
	}
	return EventAttributes{}, fmt.Errorf("kind %q is not supported", object.GetKind())
}
