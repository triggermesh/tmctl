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

package env

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	sourcesv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"

	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/awscloudwatchlogssource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/awscloudwatchsource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/awscodecommitsource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/awscognitoidentitysource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/awscognitouserpoolsource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/awsdynamodbsource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/awseventbridgesource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/awskinesissource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/awsperformanceinsightssource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/awss3source"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/awssqssource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/azureactivitylogssource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/azureblobstoragesource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/azureeventgridsource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/azureeventhubssource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/azureiothubsource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/azurequeuestoragesource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/azureservicebusqueuesource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/azureservicebustopicsource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/cloudeventssource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/googlecloudauditlogssource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/googlecloudbillingsource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/googlecloudpubsubsource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/googlecloudsourcerepositoriessource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/googlecloudstoragesource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/httppollersource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/ibmmqsource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/kafkasource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/mongodbsource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/ocimetricssource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/salesforcesource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/slacksource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/solacesource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/webhooksource"
)

func sources(object unstructured.Unstructured) ([]corev1.EnvVar, error) {
	switch object.GetKind() {
	case "AWSCloudWatchLogsSource":
		var o *sourcesv1alpha1.AWSCloudWatchLogsSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return awscloudwatchlogssource.MakeAppEnv(o), nil
	case "AWSCloudWatchSource":
		var o *sourcesv1alpha1.AWSCloudWatchSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return awscloudwatchsource.MakeAppEnv(o)
	case "AWSCodeCommitSource":
		var o *sourcesv1alpha1.AWSCodeCommitSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return awscodecommitsource.MakeAppEnv(o), nil
	case "AWSCognitoIdentitySource":
		var o *sourcesv1alpha1.AWSCognitoIdentitySource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return awscognitoidentitysource.MakeAppEnv(o), nil
	case "AWSCognitoUserPoolSource":
		var o *sourcesv1alpha1.AWSCognitoUserPoolSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return awscognitouserpoolsource.MakeAppEnv(o), nil
	case "AWSDynamoDBSource":
		var o *sourcesv1alpha1.AWSDynamoDBSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return awsdynamodbsource.MakeAppEnv(o), nil
	case "AWSEventBridgeSource":
		var o *sourcesv1alpha1.AWSEventBridgeSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return awseventbridgesource.MakeAppEnv(o), nil
	case "AWSKinesisSource":
		var o *sourcesv1alpha1.AWSKinesisSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return awskinesissource.MakeAppEnv(o), nil
	case "AWSPerformanceInsightsSource":
		var o *sourcesv1alpha1.AWSPerformanceInsightsSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return awsperformanceinsightssource.MakeAppEnv(o), nil
	case "AWSS3Source":
		var o *sourcesv1alpha1.AWSS3Source
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return awss3source.MakeAppEnv(o), nil
	case "AWSSQSSource":
		var o *sourcesv1alpha1.AWSSQSSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return awssqssource.MakeAppEnv(o), nil
	case "AzureActivityLogsSource":
		var o *sourcesv1alpha1.AzureActivityLogsSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return azureactivitylogssource.MakeAppEnv(o), nil
	case "AzureBlobStorageSource":
		var o *sourcesv1alpha1.AzureBlobStorageSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return azureblobstoragesource.MakeAppEnv(o), nil
	case "AzureEventGridSource":
		var o *sourcesv1alpha1.AzureEventGridSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return azureeventgridsource.MakeAppEnv(o), nil
	case "AzureEventHubsSource":
		var o *sourcesv1alpha1.AzureEventHubsSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return azureeventhubssource.MakeAppEnv(o), nil
	case "AzureIOTHubSource":
		var o *sourcesv1alpha1.AzureIOTHubSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return azureiothubsource.MakeAppEnv(o), nil
	case "AzureQueueStorageSource":
		var o *sourcesv1alpha1.AzureQueueStorageSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return azurequeuestoragesource.MakeAppEnv(o), nil
	case "AzureServiceBusQueueSource":
		var o *sourcesv1alpha1.AzureServiceBusQueueSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return azureservicebusqueuesource.MakeAppEnv(o), nil
	case "AzureServiceBusTopicSource":
		var o *sourcesv1alpha1.AzureServiceBusTopicSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return azureservicebustopicsource.MakeAppEnv(o), nil
	case "CloudEventsSource":
		var o *sourcesv1alpha1.CloudEventsSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return cloudeventssource.MakeAppEnv(o), nil
	case "GoogleCloudAuditLogsSource":
		var o *sourcesv1alpha1.GoogleCloudAuditLogsSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return googlecloudauditlogssource.MakeAppEnv(o), nil
	case "GoogleCloudBillingSource":
		var o *sourcesv1alpha1.GoogleCloudBillingSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return googlecloudbillingsource.MakeAppEnv(o), nil
	case "GoogleCloudPubSubSource":
		var o *sourcesv1alpha1.GoogleCloudPubSubSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return googlecloudpubsubsource.MakeAppEnv(o), nil
	case "GoogleCloudSourceRepositoriesSource":
		var o *sourcesv1alpha1.GoogleCloudSourceRepositoriesSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return googlecloudsourcerepositoriessource.MakeAppEnv(o), nil
	case "GoogleCloudStorageSource":
		var o *sourcesv1alpha1.GoogleCloudStorageSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return googlecloudstoragesource.MakeAppEnv(o), nil
	case "HTTPPollerSource":
		var o *sourcesv1alpha1.HTTPPollerSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return httppollersource.MakeAppEnv(o), nil
	case "IBMMQSource":
		var o *sourcesv1alpha1.IBMMQSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return ibmmqsource.MakeAppEnv(o), nil
	case "KafkaSource":
		var o *sourcesv1alpha1.KafkaSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return kafkasource.MakeAppEnv(o), nil
	case "MongoDBSource":
		var o *sourcesv1alpha1.MongoDBSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return mongodbsource.MakeAppEnv(o), nil
	case "OCIMetricsSource":
		var o *sourcesv1alpha1.OCIMetricsSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return ocimetricssource.MakeAppEnv(o)
	case "SalesforceSource":
		var o *sourcesv1alpha1.SalesforceSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return salesforcesource.MakeAppEnv(o), nil
	case "SlackSource":
		var o *sourcesv1alpha1.SlackSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return slacksource.MakeAppEnv(o), nil
	case "SolaceSource":
		var o *sourcesv1alpha1.SolaceSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return solacesource.MakeAppEnv(o), nil
	case "TwilioSource":
		return []corev1.EnvVar{}, nil
	case "WebhookSource":
		var o *sourcesv1alpha1.WebhookSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return webhooksource.MakeAppEnv(o), nil
	// Multitenant
	case "AWSSNSSource", "ZendeskSource":
		return nil, fmt.Errorf("kind %q is multitenant and not suitable for local environment", object.GetKind())
	}
	return nil, fmt.Errorf("kind %q is not supported", object.GetKind())
}
