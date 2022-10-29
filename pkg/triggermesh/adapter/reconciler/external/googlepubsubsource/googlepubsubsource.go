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

package googlepubsubsource

import (
	"context"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"

	sourcesv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
)

func Client(src *sourcesv1alpha1.GoogleCloudPubSubSource, secrets map[string]string) (*pubsub.Client, error) {
	project := src.Spec.Topic.Project
	saKey := []byte{}
	for _, v := range secrets {
		saKey = []byte(v)
		break
	}
	return pubsub.NewClient(context.Background(), project, option.WithCredentialsJSON(saKey))
}
