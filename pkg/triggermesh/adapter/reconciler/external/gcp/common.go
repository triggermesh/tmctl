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

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"

	sourcesv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
)

func pubSubClient(ctx context.Context, spec sourcesv1alpha1.GoogleCloudSourcePubSubSpec, secrets map[string]string) (*pubsub.Client, option.ClientOption, error) {
	saKey, exists := secrets["serviceAccountKey"]
	if !exists {
		return nil, nil, fmt.Errorf("\"serviceAccountKey\" is missing")
	}
	credsCliOpt := option.WithCredentialsJSON([]byte(saKey))

	var pubsubProject string
	if project := spec.Project; project != nil {
		pubsubProject = *project
	} else if topic := spec.Topic; topic != nil {
		pubsubProject = topic.Project
	}
	psCli, err := pubsub.NewClient(ctx, pubsubProject, credsCliOpt)
	return psCli, credsCliOpt, fmt.Errorf("creating Google Cloud Pub/Sub API client: %w", err)
}
