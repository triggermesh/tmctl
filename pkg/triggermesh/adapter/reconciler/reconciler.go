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
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	sourcesv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/awss3source"

	"github.com/triggermesh/tmctl/pkg/triggermesh/adapter/reconciler/fake"
)

func InitializeAndGetStatus(ctx context.Context, object unstructured.Unstructured, secrets map[string]string) (map[string]interface{}, error) {
	switch object.GetKind() {
	case "AWSS3Source":
		var o *sourcesv1alpha1.AWSS3Source
		accessKey, secretKey, err := fake.ReadSecret(secrets)
		if err != nil {
			return nil, err
		}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		s3Client, sqsClient, err := fake.NewAWSS3ClientGetter(accessKey, secretKey).Get(o)
		if err != nil {
			return nil, err
		}
		arn, err := awss3source.EnsureQueue(ctx, sqsClient)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"queueARN": arn}, awss3source.EnsureNotificationsEnabled(ctx, s3Client, arn)
	}
	return nil, nil
}

func Finalize(ctx context.Context, object unstructured.Unstructured, secrets map[string]string) error {
	switch object.GetKind() {
	case "AWSS3Source":
		var o *sourcesv1alpha1.AWSS3Source
		accessKey, secretKey, err := fake.ReadSecret(secrets)
		if err != nil {
			return err
		}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return err
		}
		ctx = commonv1alpha1.WithReconcilable(ctx, o)
		s3Client, sqsClient, err := fake.NewAWSS3ClientGetter(accessKey, secretKey).Get(o)
		if err != nil {
			return err
		}
		if err := awss3source.EnsureNoQueue(ctx, sqsClient); err != nil {
			return err
		}
		if err := awss3source.EnsureNotificationsDisabled(ctx, s3Client); strings.Contains(strings.ToLower(err.Error()), "error") {
			// EnsureNotificationsDisabled returns error even if it succeeded
			return err
		}
	}
	return nil
}
