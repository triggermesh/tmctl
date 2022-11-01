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

package azure

import (
	"github.com/Azure/azure-sdk-for-go/profiles/latest/eventgrid/mgmt/eventgrid"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/eventhub/mgmt/eventhub"
	"github.com/Azure/go-autorest/autorest"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/client/azure/storage"
)

func BlobStorage(src *v1alpha1.AzureBlobStorageSource, authorizer autorest.Authorizer) (storage.EventSubscriptionsClient, storage.EventHubsClient) {
	eventSubsCli := eventgrid.NewEventSubscriptionsClient(src.Spec.StorageAccountID.SubscriptionID)
	eventSubsCli.Authorizer = authorizer
	eventHubsCli := eventhub.NewEventHubsClient(src.Spec.StorageAccountID.SubscriptionID)
	eventHubsCli.Authorizer = authorizer
	return eventSubsCli, eventHubsCli
}
