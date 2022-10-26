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

package fake

import (
	"fmt"

	awscore "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sts"

	sourcesv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
)

const defaultS3Region = "us-east-1"

const (
	awsAccessKeyEnv = "accessKeyID"
	awsSecretKeyEnv = "secretAccessKey"
)

type AWSS3ClientGetter struct {
	AccessKeyID     string
	SecretAccessKey string
}

func ReadSecret(secrets map[string]string) (string, string, error) {
	accessKey, exists := secrets[awsAccessKeyEnv]
	if !exists {
		return "", "", fmt.Errorf("%q secret is missing", awsAccessKeyEnv)
	}
	secretKey, exists := secrets[awsSecretKeyEnv]
	if !exists {
		return "", "", fmt.Errorf("%q secret is missing", awsSecretKeyEnv)
	}
	return accessKey, secretKey, nil
}

func NewAWSS3ClientGetter(accessKeyID, secretAccessKey string) AWSS3ClientGetter {
	return AWSS3ClientGetter{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
	}
}

func (c AWSS3ClientGetter) Get(src *sourcesv1alpha1.AWSS3Source) (*s3.S3, *sqs.SQS, error) {
	sess := session.Must(session.NewSession(awscore.NewConfig()))

	creds := &credentials.Value{
		AccessKeyID:     c.AccessKeyID,
		SecretAccessKey: c.SecretAccessKey,
	}

	region, err := determineS3Region(src, creds)
	if err != nil {
		return nil, nil, fmt.Errorf("determining suitable S3 region: %w", err)
	}
	if src.Spec.ARN.Region == "" {
		src.Spec.ARN.Region = region
	}

	accID, err := determineBucketOwnerAccount(src, creds)
	if err != nil {
		return nil, nil, fmt.Errorf("determining bucket's owner: %w", err)
	}
	if src.Spec.ARN.AccountID == "" {
		src.Spec.ARN.AccountID = accID
	}

	sess.Config.
		WithRegion(src.Spec.ARN.Region).
		WithCredentials(credentials.NewStaticCredentialsFromCreds(*creds))

	return s3.New(sess), sqs.New(sess), nil
}

func determineS3Region(src *sourcesv1alpha1.AWSS3Source, creds *credentials.Value) (string, error) {
	if src.Spec.ARN.Region != "" {
		return src.Spec.ARN.Region, nil
	}

	if dest := src.Spec.Destination; dest != nil {
		if sqsDest := dest.SQS; sqsDest != nil {
			return sqsDest.QueueARN.Region, nil
		}
	}

	region, err := getBucketRegion(src.Spec.ARN.Resource, creds)
	if err != nil {
		return "", fmt.Errorf("getting location of bucket %q: %w", src.Spec.ARN.Resource, err)
	}

	return region, nil
}

// getBucketRegion retrieves the region the provided bucket resides in.
func getBucketRegion(bucketName string, creds *credentials.Value) (string, error) {
	sess := session.Must(session.NewSession(awscore.NewConfig().
		WithRegion(defaultS3Region).
		WithCredentials(credentials.NewStaticCredentialsFromCreds(*creds)),
	))

	resp, err := s3.New(sess).GetBucketLocation(&s3.GetBucketLocationInput{
		Bucket: &bucketName,
	})
	if err != nil {
		return "", err
	}

	if loc := resp.LocationConstraint; loc != nil {
		return *loc, nil
	}
	return defaultS3Region, nil
}

func determineBucketOwnerAccount(src *sourcesv1alpha1.AWSS3Source, creds *credentials.Value) (string, error) {
	if src.Spec.ARN.AccountID != "" {
		return src.Spec.ARN.AccountID, nil
	}

	if dest := src.Spec.Destination; dest != nil {
		if sqsDest := dest.SQS; sqsDest != nil {
			return sqsDest.QueueARN.AccountID, nil
		}
	}

	accID, err := getCallerAccountID(creds)
	if err != nil {
		return "", fmt.Errorf("getting ID of caller: %w", err)
	}

	return accID, nil
}

// getCallerAccountID retrieves the account ID of the caller.
func getCallerAccountID(creds *credentials.Value) (string, error) {
	sess := session.Must(session.NewSession(awscore.NewConfig().
		WithCredentials(credentials.NewStaticCredentialsFromCreds(*creds)),
	))

	resp, err := sts.New(sess).GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}

	return *resp.Account, nil
}
