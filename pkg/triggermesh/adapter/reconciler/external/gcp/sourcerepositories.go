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
	"google.golang.org/api/sourcerepo/v1"

	sourcesv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
)

func SourceRepoClient(ctx context.Context, src *sourcesv1alpha1.GoogleCloudSourceRepositoriesSource, secrets map[string]string) (*pubsub.Client, *sourcerepo.Service, error) {
	psCli, credsCliOpt, err := pubSubClient(ctx, src.Spec.PubSub, secrets)
	if err != nil {
		return nil, nil, err
	}
	stCli, err := sourcerepo.NewService(ctx, credsCliOpt)
	if err != nil {
		return nil, nil, fmt.Errorf("creating Google Cloud Source Repositories API client: %w", err)
	}
	return psCli, stCli, nil
}
