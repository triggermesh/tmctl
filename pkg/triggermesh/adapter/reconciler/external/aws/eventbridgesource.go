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

package aws

import (
	"fmt"

	awscore "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eventbridge"
	"github.com/aws/aws-sdk-go/service/sqs"

	sourcesv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
)

func EBClient(src *sourcesv1alpha1.AWSEventBridgeSource, secrets map[string]string) (*eventbridge.EventBridge, *sqs.SQS, error) {
	accessKey, secretKey, err := readSecret(secrets)
	if err != nil {
		return nil, nil, fmt.Errorf("secrets read: %w", err)
	}
	creds := &credentials.Value{
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
	}

	sess := session.Must(session.NewSession(awscore.NewConfig().
		WithRegion(src.Spec.ARN.Region).
		WithCredentials(credentials.NewStaticCredentialsFromCreds(*creds)),
	))

	return eventbridge.New(sess), sqs.New(sess), nil
}
