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

package gcp

import (
	"context"
	"fmt"

	billing "cloud.google.com/go/billing/budgets/apiv1"
	"cloud.google.com/go/pubsub"
	sourcesv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
)

func CloudBillingClient(src *sourcesv1alpha1.GoogleCloudBillingSource, secrets map[string]string) (*pubsub.Client, *billing.BudgetClient, error) {
	ctx := context.Background()
	psClient, credsClientOptions, err := pubSubClient(ctx, src.Spec.PubSub, secrets)
	if err != nil {
		return nil, nil, err
	}
	biClient, err := billing.NewBudgetClient(ctx, credsClientOptions)
	if err != nil {
		return nil, nil, fmt.Errorf("creating Google Cloud Billing Budget API client: %w", err)
	}
	return psClient, biClient, nil
}
