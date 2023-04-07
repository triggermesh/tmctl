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
	tmawseb "github.com/triggermesh/triggermesh/pkg/sources/reconciler/awseventbridgesource"
	tmawss3 "github.com/triggermesh/triggermesh/pkg/sources/reconciler/awss3source"
	tmazureblobstorage "github.com/triggermesh/triggermesh/pkg/sources/reconciler/azureblobstoragesource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/azureeventgridsource"
	tmazureservicebus "github.com/triggermesh/triggermesh/pkg/sources/reconciler/azureservicebustopicsource"
	tmgcpauditlogs "github.com/triggermesh/triggermesh/pkg/sources/reconciler/googlecloudauditlogssource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/googlecloudbillingsource"
	tmgcppubsub "github.com/triggermesh/triggermesh/pkg/sources/reconciler/googlecloudpubsubsource"
	tmgcprepo "github.com/triggermesh/triggermesh/pkg/sources/reconciler/googlecloudsourcerepositoriessource"
	tmgcpstorage "github.com/triggermesh/triggermesh/pkg/sources/reconciler/googlecloudstoragesource"

	"github.com/triggermesh/tmctl/pkg/triggermesh/adapter/reconciler/external/aws"
	"github.com/triggermesh/tmctl/pkg/triggermesh/adapter/reconciler/external/azure"
	"github.com/triggermesh/tmctl/pkg/triggermesh/adapter/reconciler/external/gcp"
)

func InitializeAndGetStatus(ctx context.Context, object unstructured.Unstructured, secrets map[string]string) (map[string]interface{}, error) {
	switch object.GetKind() {
	case "AWSS3Source":
		var o *sourcesv1alpha1.AWSS3Source
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		s3Client, sqsClient, err := aws.S3Client(o, secrets)
		if err != nil {
			return nil, err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		arn, err := tmawss3.EnsureQueue(ctx, sqsClient)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"queueARN": arn}, tmawss3.EnsureNotificationsEnabled(ctx, s3Client, arn)
	case "AWSEventBridgeSource":
		var o *sourcesv1alpha1.AWSEventBridgeSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		ebClient, sqsClient, err := aws.EBClient(o, secrets)
		if err != nil {
			return nil, err
		}
		queue, err := tmawseb.EnsureQueue(ctx, sqsClient)
		if err != nil {
			return nil, err
		}
		ruleARN, err := tmawseb.EnsureRule(ctx, ebClient, queue)
		if err != nil {
			return nil, err
		}
		if err := tmawseb.EnsureQueuePolicy(ctx, sqsClient, queue, ruleARN); err != nil {
			return nil, err
		}
		if err := tmawseb.SetRuleTarget(ctx, ebClient, ruleARN, queue.ARN); err != nil {
			return nil, err
		}
		return map[string]interface{}{"queueARN": queue.ARN}, nil
	case "AzureEventGridSource":
		var o *sourcesv1alpha1.AzureEventGridSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		authorizer, err := azure.Client(secrets)
		if err != nil {
			return nil, err
		}
		sysTopicsCli, providersCli, resGroupsCli, eventSubsCli, eventHubsCli := azure.EventGridClient(o, authorizer)
		sysTopicResID, err := azureeventgridsource.EnsureSystemTopic(ctx, sysTopicsCli, providersCli, resGroupsCli)
		if err != nil {
			return nil, err
		}
		eventHubResID, err := azureeventgridsource.EnsureEventHub(ctx, eventHubsCli)
		if err != nil {
			return nil, err
		}
		if err := azureeventgridsource.EnsureEventSubscription(ctx, eventSubsCli, sysTopicResID, eventHubResID); err != nil {
			return nil, err
		}
		return map[string]interface{}{"subscriptionID": o.Status.EventSubscriptionID.String()}, nil
	case "AzureServiceBusTopicSource":
		var o *sourcesv1alpha1.AzureServiceBusTopicSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		authorizer, err := azure.Client(secrets)
		if err != nil {
			return nil, err
		}
		client := azure.ServiceBus(o, authorizer)
		if err := tmazureservicebus.EnsureSubscription(ctx, client); err != nil {
			return nil, err
		}
		return map[string]interface{}{"subscriptionID": o.Status.SubscriptionID.String()}, nil
	case "AzureBlobStorageSource":
		var o *sourcesv1alpha1.AzureBlobStorageSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		authorizer, err := azure.Client(secrets)
		if err != nil {
			return nil, err
		}
		subsAPI, hubsAPI := azure.BlobStorage(o, authorizer)
		eventHubID, err := tmazureblobstorage.EnsureEventHub(ctx, hubsAPI)
		if err != nil {
			return nil, err
		}
		if err := tmazureblobstorage.EnsureEventSubscription(ctx, subsAPI, eventHubID); err != nil {
			return nil, err
		}
		return map[string]interface{}{"eventHubID": eventHubID}, nil
	case "GoogleCloudBillingSource":
		var o *sourcesv1alpha1.GoogleCloudBillingSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		psClient, billingClient, err := gcp.CloudBillingClient(o, secrets)
		if err != nil {
			return nil, err
		}
		topic, err := googlecloudbillingsource.EnsurePubSub(ctx, psClient)
		if err != nil {
			return nil, err
		}
		if err := googlecloudbillingsource.EnsureBudgetNotification(ctx, billingClient, topic); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"topic":        o.Status.Topic,
			"subscription": o.Status.Subscription,
		}, nil
	case "GoogleCloudPubSubSource":
		var o *sourcesv1alpha1.GoogleCloudPubSubSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		client, err := gcp.PubSubClient(o, secrets)
		if err != nil {
			return nil, err
		}
		if err := tmgcppubsub.EnsureSubscription(ctx, client); err != nil {
			return nil, err
		}
		return map[string]interface{}{"subscription": o.Status.Subscription.String()}, nil
	case "GoogleCloudAuditLogsSource":
		var o *sourcesv1alpha1.GoogleCloudAuditLogsSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		psCli, laCli, err := gcp.AuditLogsClient(ctx, o, secrets)
		if err != nil {
			return nil, err
		}
		topic, err := tmgcpauditlogs.EnsurePubSub(ctx, psCli)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"subscription": o.Status.Subscription.String()}, tmgcpauditlogs.ReconcileSink(ctx, laCli, psCli, topic)
	case "GoogleCloudStorageSource":
		var o *sourcesv1alpha1.GoogleCloudStorageSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		psCli, stCli, err := gcp.StorageClient(ctx, o, secrets)
		if err != nil {
			return nil, err
		}

		topic, err := tmgcpstorage.EnsurePubSub(ctx, psCli)
		if err != nil {
			return nil, err
		}
		if err := tmgcpstorage.EnsureNotificationConfig(ctx, stCli, topic); err != nil {
			return nil, err
		}
		return map[string]interface{}{"subscription": o.Status.Subscription.String()}, tmgcpstorage.EnsureNotificationConfig(ctx, stCli, topic)
	case "GoogleCloudSourceRepositoriesSource":
		var o *sourcesv1alpha1.GoogleCloudSourceRepositoriesSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		psCli, repoCli, err := gcp.SourceRepoClient(ctx, o, secrets)
		if err != nil {
			return nil, err
		}
		topic, err := tmgcprepo.EnsurePubSub(ctx, psCli)
		if err != nil {
			return nil, err
		}
		var publishServiceAccount string
		if sa := o.Spec.PublishServiceAccount; sa != nil {
			publishServiceAccount = *sa
		}
		return map[string]interface{}{"subscription": o.Status.Subscription.String()}, tmgcprepo.EnsureTopicAssociated(ctx, repoCli, topic, publishServiceAccount)

	case "AzureActivityLogsSource":
		return nil, fmt.Errorf("this component is not suitable for local env yet")
	case "ZendeskSource":
		return nil, fmt.Errorf("this component is multitenant and not suitable for local env")
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
		s3Client, sqsClient, err := aws.S3Client(o, secrets)
		if err != nil {
			return err
		}
		if err := tmawss3.EnsureNoQueue(ctx, sqsClient); err != nil {
			return err
		}
		if err := tmawss3.EnsureNotificationsDisabled(ctx, s3Client); strings.Contains(strings.ToLower(err.Error()), "error") {
			// EnsureNotificationsDisabled returns "normal" event (error) if it succeeded
			return err
		}
	case "AWSEventBridgeSource":
		var o *sourcesv1alpha1.AWSEventBridgeSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		ebClient, sqsClient, err := aws.EBClient(o, secrets)
		if err != nil {
			return err
		}
		queueName, err := tmawseb.EnsureNoRule(ctx, ebClient, sqsClient)
		if err != nil {
			return err
		}
		if err := tmawseb.EnsureNoQueue(ctx, sqsClient, queueName); err != nil {
			return err
		}
	case "AzureEventGridSource":
		var o *sourcesv1alpha1.AzureEventGridSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		authorizer, err := azure.Client(secrets)
		if err != nil {
			return err
		}
		sysTopicsCli, _, _, eventSubsCli, eventHubsCli := azure.EventGridClient(o, authorizer)
		systemTopic, err := azureeventgridsource.FindSystemTopic(ctx, sysTopicsCli, o)
		if err != nil {
			return err
		}
		if err := azureeventgridsource.EnsureNoEventSubscription(ctx, eventSubsCli, systemTopic); err != nil {
			return err
		}
		if err := azureeventgridsource.EnsureNoEventHub(ctx, eventHubsCli); err != nil {
			return err
		}
		return azureeventgridsource.EnsureNoSystemTopic(ctx, sysTopicsCli, eventSubsCli, systemTopic)
	case "AzureServiceBusTopicSource":
		var o *sourcesv1alpha1.AzureServiceBusTopicSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		authorizer, err := azure.Client(secrets)
		if err != nil {
			return err
		}
		client := azure.ServiceBus(o, authorizer)
		return tmazureservicebus.EnsureNoSubscription(ctx, client)
	case "AzureBlobStorageSource":
		var o *sourcesv1alpha1.AzureBlobStorageSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		authorizer, err := azure.Client(secrets)
		if err != nil {
			return err
		}
		subsAPI, hubsAPI := azure.BlobStorage(o, authorizer)
		if err := tmazureblobstorage.EnsureNoEventHub(ctx, hubsAPI); err != nil {
			return err
		}
		return tmazureblobstorage.EnsureNoEventSubscription(ctx, subsAPI)
	case "GoogleCloudBillingSource":
		var o *sourcesv1alpha1.GoogleCloudBillingSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		psClient, billingClient, err := gcp.CloudBillingClient(o, secrets)
		if err != nil {
			return err
		}
		if err := googlecloudbillingsource.EnsureNoBudgetNotification(ctx, billingClient); err != nil {
			return err
		}
		return googlecloudbillingsource.EnsureNoPubSub(ctx, psClient)
	case "GoogleCloudPubSubSource":
		var o *sourcesv1alpha1.GoogleCloudPubSubSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		client, err := gcp.PubSubClient(o, secrets)
		if err != nil {
			return err
		}
		err = tmgcppubsub.EnsureNoSubscription(ctx, client) // err is never nil
		if err.Error() != fmt.Sprintf("Unsubscribed from topic %q", o.Spec.Topic) {
			return err
		}
	case "GoogleCloudAuditLogsSource":
		var o *sourcesv1alpha1.GoogleCloudAuditLogsSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		psCli, laCli, err := gcp.AuditLogsClient(ctx, o, secrets)
		if err != nil {
			return err
		}
		if err := tmgcpauditlogs.EnsureNoSink(ctx, laCli); err != nil {
			return err
		}
		return tmgcpauditlogs.EnsureNoPubSub(ctx, psCli)
	case "GoogleCloudStorageSource":
		var o *sourcesv1alpha1.GoogleCloudStorageSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		psCli, stCli, err := gcp.StorageClient(ctx, o, secrets)
		if err != nil {
			return err
		}
		if err := tmgcpstorage.EnsureNoNotificationConfig(ctx, stCli); err != nil {
			return err
		}
		return tmgcpstorage.EnsureNoPubSub(ctx, psCli)
	case "GoogleCloudSourceRepositoriesSource":
		var o *sourcesv1alpha1.GoogleCloudSourceRepositoriesSource
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		psCli, repoCli, err := gcp.SourceRepoClient(ctx, o, secrets)
		if err != nil {
			return err
		}
		if err := tmgcprepo.EnsureNoTopicAssociated(ctx, repoCli); err != nil {
			return err
		}
		return tmgcprepo.EnsureNoPubSub(ctx, psCli)

	case "AzureActivityLogsSource":
		return fmt.Errorf("this component is not suitable for local env yet")
	case "ZendeskSource":
		return fmt.Errorf("this component is multitenant and not suitable for local env")
	}
	return nil
}
