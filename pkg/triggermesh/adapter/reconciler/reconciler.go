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

package reconciler

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	sourcesv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	externalawseb "github.com/triggermesh/triggermesh/pkg/sources/reconciler/awseventbridgesource"
	externalawss3 "github.com/triggermesh/triggermesh/pkg/sources/reconciler/awss3source"
	externalazureservicebus "github.com/triggermesh/triggermesh/pkg/sources/reconciler/azureservicebustopicsource"
	externalpubsub "github.com/triggermesh/triggermesh/pkg/sources/reconciler/googlecloudpubsubsource"

	"github.com/triggermesh/tmctl/pkg/triggermesh/adapter/reconciler/external/awseventbridgesource"
	"github.com/triggermesh/tmctl/pkg/triggermesh/adapter/reconciler/external/awss3source"
	"github.com/triggermesh/tmctl/pkg/triggermesh/adapter/reconciler/external/azureservicebustopicsource"
	"github.com/triggermesh/tmctl/pkg/triggermesh/adapter/reconciler/external/googlepubsubsource"
)

func InitializeAndGetStatus(ctx context.Context, object unstructured.Unstructured, secrets map[string]string) (map[string]interface{}, error) {
	switch object.GetKind() {
	case "AWSS3Source":
		var o *sourcesv1alpha1.AWSS3Source
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		s3Client, sqsClient, err := awss3source.Client(o, secrets)
		if err != nil {
			return nil, err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		arn, err := externalawss3.EnsureQueue(ctx, sqsClient)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"queueARN": arn}, externalawss3.EnsureNotificationsEnabled(ctx, s3Client, arn)
	case "AWSEventBridgeSource":
		var o *sourcesv1alpha1.AWSEventBridgeSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		ebClient, sqsClient, err := awseventbridgesource.Client(o, secrets)
		if err != nil {
			return nil, err
		}
		queue, err := externalawseb.EnsureQueue(ctx, sqsClient)
		if err != nil {
			return nil, err
		}
		ruleARN, err := externalawseb.EnsureRule(ctx, ebClient, queue)
		if err != nil {
			return nil, err
		}
		if err := externalawseb.EnsureQueuePolicy(ctx, sqsClient, queue, ruleARN); err != nil {
			return nil, err
		}
		if err := externalawseb.SetRuleTarget(ctx, ebClient, ruleARN, queue.ARN); err != nil {
			return nil, err
		}
		return map[string]interface{}{"queueARN": queue.ARN}, nil
	case "GoogleCloudPubSubSource":
		var o *sourcesv1alpha1.GoogleCloudPubSubSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		client, err := googlepubsubsource.Client(o, secrets)
		if err != nil {
			return nil, err
		}
		if err := externalpubsub.EnsureSubscription(ctx, client); err != nil {
			return nil, err
		}
		return map[string]interface{}{"subscription": o.Status.Subscription.String()}, nil
	case "AzureServiceBusTopicSource":
		var o *sourcesv1alpha1.AzureServiceBusTopicSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		client, err := azureservicebustopicsource.Client(o, secrets)
		if err != nil {
			return nil, err
		}
		if err := externalazureservicebus.EnsureSubscription(ctx, client); err != nil {
			return nil, err
		}
		return map[string]interface{}{"subscriptionID": o.Status.SubscriptionID}, nil
	case "AzureActivityLogsSource",
		"AzureBlobStorageSource",
		"AzureEventGridSource",
		"GoogleCloudAuditLogsSource",
		"GoogleCloudBillingSource",
		"GoogleCloudIoTSource",
		"GoogleCloudSourceRepositoriesSource",
		"GoogleCloudStorageSource",
		"ZendeskSource":
		return nil, fmt.Errorf("this component is not suitable for local env yet")
	}
	return nil, nil
}

func Finalize(ctx context.Context, object unstructured.Unstructured, secrets map[string]string) error {
	switch object.GetKind() {
	case "AWSS3Source":
		var o *sourcesv1alpha1.AWSS3Source
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		s3Client, sqsClient, err := awss3source.Client(o, secrets)
		if err != nil {
			return err
		}
		if err := externalawss3.EnsureNoQueue(ctx, sqsClient); err != nil {
			return err
		}
		if err := externalawss3.EnsureNotificationsDisabled(ctx, s3Client); strings.Contains(strings.ToLower(err.Error()), "error") {
			// EnsureNotificationsDisabled returns "normal" event (error) if it succeeded
			return err
		}
	case "AWSEventBridgeSource":
		var o *sourcesv1alpha1.AWSEventBridgeSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		ebClient, sqsClient, err := awseventbridgesource.Client(o, secrets)
		if err != nil {
			return err
		}
		queueName, err := externalawseb.EnsureNoRule(ctx, ebClient, sqsClient)
		if err != nil {
			return err
		}
		if err := externalawseb.EnsureNoQueue(ctx, sqsClient, queueName); err != nil {
			return err
		}
	case "GoogleCloudPubSubSource":
		var o *sourcesv1alpha1.GoogleCloudPubSubSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		client, err := googlepubsubsource.Client(o, secrets)
		if err != nil {
			return err
		}
		err = externalpubsub.EnsureNoSubscription(ctx, client) // err is never nil
		if err.Error() != fmt.Sprintf("Unsubscribed from topic %q", o.Spec.Topic) {
			return err
		}
	case "AzureServiceBusTopicSource":
		var o *sourcesv1alpha1.AzureServiceBusTopicSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		client, err := azureservicebustopicsource.Client(o, secrets)
		if err != nil {
			return err
		}
		return externalazureservicebus.EnsureNoSubscription(ctx, client)
	case "AzureActivityLogsSource",
		"AzureBlobStorageSource",
		"AzureEventGridSource",
		"GoogleCloudAuditLogsSource",
		"GoogleCloudBillingSource",
		"GoogleCloudIoTSource",
		"GoogleCloudSourceRepositoriesSource",
		"GoogleCloudStorageSource",
		"ZendeskSource":
		return fmt.Errorf("this component is not suitable for local env yet")
	}
	return nil
}
