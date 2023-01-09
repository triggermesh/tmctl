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

	sourcesv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
)

func sources(object unstructured.Unstructured) (EventAttributes, error) {
	switch object.GetKind() {
	case "AWSCloudWatchLogsSource":
		var o *sourcesv1alpha1.AWSCloudWatchLogsSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "AWSCloudWatchSource":
		var o *sourcesv1alpha1.AWSCloudWatchSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "AWSCodeCommitSource":
		var o *sourcesv1alpha1.AWSCodeCommitSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "AWSCognitoIdentitySource":
		var o *sourcesv1alpha1.AWSCognitoIdentitySource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "AWSCognitoUserPoolSource":
		var o *sourcesv1alpha1.AWSCognitoUserPoolSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "AWSDynamoDBSource":
		var o *sourcesv1alpha1.AWSDynamoDBSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "AWSEventBridgeSource":
		var o *sourcesv1alpha1.AWSEventBridgeSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "AWSKinesisSource":
		var o *sourcesv1alpha1.AWSKinesisSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "AWSPerformanceInsightsSource":
		var o *sourcesv1alpha1.AWSPerformanceInsightsSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "AWSS3Source":
		var o *sourcesv1alpha1.AWSS3Source
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "AWSSQSSource":
		var o *sourcesv1alpha1.AWSSQSSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "AzureActivityLogsSource":
		var o *sourcesv1alpha1.AzureActivityLogsSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "AzureBlobStorageSource":
		var o *sourcesv1alpha1.AzureBlobStorageSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "AzureEventGridSource":
		var o *sourcesv1alpha1.AzureEventGridSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "AzureEventHubsSource":
		var o *sourcesv1alpha1.AzureEventHubsSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "AzureIOTHubSource":
		var o *sourcesv1alpha1.AzureIOTHubSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "AzureQueueStorageSource":
		var o *sourcesv1alpha1.AzureQueueStorageSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "AzureServiceBusQueueSource":
		var o *sourcesv1alpha1.AzureServiceBusQueueSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "AzureServiceBusTopicSource":
		var o *sourcesv1alpha1.AzureServiceBusTopicSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "CloudEventsSource":
		var o *sourcesv1alpha1.CloudEventsSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "GoogleCloudAuditLogsSource":
		var o *sourcesv1alpha1.GoogleCloudAuditLogsSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "GoogleCloudBillingSource":
		var o *sourcesv1alpha1.GoogleCloudBillingSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "GoogleCloudPubSubSource":
		var o *sourcesv1alpha1.GoogleCloudPubSubSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "GoogleCloudSourceRepositoriesSource":
		var o *sourcesv1alpha1.GoogleCloudSourceRepositoriesSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "GoogleCloudStorageSource":
		var o *sourcesv1alpha1.GoogleCloudStorageSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "HTTPPollerSource":
		var o *sourcesv1alpha1.HTTPPollerSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "IBMMQSource":
		var o *sourcesv1alpha1.IBMMQSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "KafkaSource":
		var o *sourcesv1alpha1.KafkaSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "OCIMetricsSource":
		var o *sourcesv1alpha1.OCIMetricsSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "SalesforceSource":
		var o *sourcesv1alpha1.SalesforceSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "SlackSource":
		var o *sourcesv1alpha1.SlackSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "TwilioSource":
		var o *sourcesv1alpha1.TwilioSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	case "WebhookSource":
		var o *sourcesv1alpha1.WebhookSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return EventAttributes{}, err
		}
		return EventAttributes{
			ProducedEventTypes:  o.GetEventTypes(),
			ProducedEventSource: o.AsEventSource(),
		}, nil
	// Multitenant
	case "AWSSNSSource", "ZendeskSource":
		return EventAttributes{}, fmt.Errorf("kind %q is multitenant and not suitable for local environment", object.GetKind())
	}

	return EventAttributes{}, fmt.Errorf("kind %q is not supported", object.GetKind())
}
