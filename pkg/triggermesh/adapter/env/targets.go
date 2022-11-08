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

package env

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	targetsv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"

	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/alibabaosstarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/awscomprehendtarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/awsdynamodbtarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/awseventbridgetarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/awskinesistarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/awslambdatarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/awss3target"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/awssnstarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/awssqstarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/azureeventhubstarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/cloudeventstarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/confluenttarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/datadogtarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/elasticsearchtarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/googlecloudfirestoretarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/googlecloudpubsubtarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/googlecloudstoragetarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/googlecloudworkflowstarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/googlesheettarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/hasuratarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/httptarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/ibmmqtarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/infratarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/jiratarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/kafkatarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/logzmetricstarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/logztarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/oracletarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/salesforcetarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/sendgridtarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/slacktarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/splunktarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/tektontarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/twiliotarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/uipathtarget"
	"github.com/triggermesh/triggermesh/pkg/targets/reconciler/zendesktarget"
)

func targets(object unstructured.Unstructured) ([]corev1.EnvVar, error) {
	switch object.GetKind() {
	// Flow API group
	case "AlibabaOSSTarget":
		var o *targetsv1alpha1.AlibabaOSSTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return alibabaosstarget.MakeAppEnv(o), nil
	case "AWSComprehendTarget":
		var o *targetsv1alpha1.AWSComprehendTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return awscomprehendtarget.MakeAppEnv(o), nil
	case "AWSDynamoDBTarget":
		var o *targetsv1alpha1.AWSDynamoDBTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return awsdynamodbtarget.MakeAppEnv(o), nil
	case "AWSEventBridgeTarget":
		var o *targetsv1alpha1.AWSEventBridgeTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return awseventbridgetarget.MakeAppEnv(o), nil
	case "AWSKinesisTarget":
		var o *targetsv1alpha1.AWSKinesisTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return awskinesistarget.MakeAppEnv(o), nil
	case "AWSLambdaTarget":
		var o *targetsv1alpha1.AWSLambdaTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return awslambdatarget.MakeAppEnv(o), nil
	case "AWSS3Target":
		var o *targetsv1alpha1.AWSS3Target
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return awss3target.MakeAppEnv(o), nil
	case "AWSSNSTarget":
		var o *targetsv1alpha1.AWSSNSTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return awssnstarget.MakeAppEnv(o), nil
	case "AWSSQSTarget":
		var o *targetsv1alpha1.AWSSQSTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return awssqstarget.MakeAppEnv(o), nil
	case "AzureEventHubsTarget":
		var o *targetsv1alpha1.AzureEventHubsTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return azureeventhubstarget.MakeAppEnv(o), nil
	case "CloudEventsTarget":
		var o *targetsv1alpha1.CloudEventsTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return cloudeventstarget.MakeAppEnv(o), nil
	case "ConfluentTarget":
		var o *targetsv1alpha1.ConfluentTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return confluenttarget.MakeAppEnv(o), nil
	case "DatadogTarget":
		var o *targetsv1alpha1.DatadogTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return datadogtarget.MakeAppEnv(o), nil
	case "ElasticsearchTarget":
		var o *targetsv1alpha1.ElasticsearchTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return elasticsearchtarget.MakeAppEnv(o), nil
	case "GoogleCloudFirestoreTarget":
		var o *targetsv1alpha1.GoogleCloudFirestoreTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return googlecloudfirestoretarget.MakeAppEnv(o), nil
	case "GoogleCloudPubSubTarget":
		var o *targetsv1alpha1.GoogleCloudPubSubTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return googlecloudpubsubtarget.MakeAppEnv(o), nil
	case "GoogleCloudStorageTarget":
		var o *targetsv1alpha1.GoogleCloudStorageTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return googlecloudstoragetarget.MakeAppEnv(o), nil
	case "GoogleCloudWorkflowsTarget":
		var o *targetsv1alpha1.GoogleCloudWorkflowsTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return googlecloudworkflowstarget.MakeAppEnv(o), nil
	case "GoogleSheetTarget":
		var o *targetsv1alpha1.GoogleSheetTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return googlesheettarget.MakeAppEnv(o), nil
	case "HasuraTarget":
		var o *targetsv1alpha1.HasuraTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return hasuratarget.MakeAppEnv(o), nil
	case "HTTPTarget":
		var o *targetsv1alpha1.HTTPTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return httptarget.MakeAppEnv(o), nil
	case "IBMMQTarget":
		var o *targetsv1alpha1.IBMMQTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return ibmmqtarget.MakeAppEnv(o), nil
	case "InfraTarget":
		var o *targetsv1alpha1.InfraTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return infratarget.MakeAppEnv(o), nil
	case "JiraTarget":
		var o *targetsv1alpha1.JiraTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return jiratarget.MakeAppEnv(o), nil
	case "KafkaTarget":
		var o *targetsv1alpha1.KafkaTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return kafkatarget.MakeAppEnv(o), nil
	case "LogzMetricsTarget":
		var o *targetsv1alpha1.LogzMetricsTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return logzmetricstarget.MakeAppEnv(o), nil
	case "LogzTarget":
		var o *targetsv1alpha1.LogzTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return logztarget.MakeAppEnv(o), nil
	case "OracleTarget":
		var o *targetsv1alpha1.OracleTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return oracletarget.MakeAppEnv(o), nil
	case "SalesforceTarget":
		var o *targetsv1alpha1.SalesforceTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return salesforcetarget.MakeAppEnv(o), nil
	case "SendGridTarget":
		var o *targetsv1alpha1.SendGridTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return sendgridtarget.MakeAppEnv(o), nil
	case "SlackTarget":
		var o *targetsv1alpha1.SlackTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return slacktarget.MakeAppEnv(o), nil
	case "SplunkTarget":
		var o *targetsv1alpha1.SplunkTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return splunktarget.MakeAppEnv(o), nil
	case "TektonTarget":
		var o *targetsv1alpha1.TektonTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return tektontarget.MakeAppEnv(o), nil
	case "TwilioTarget":
		var o *targetsv1alpha1.TwilioTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return twiliotarget.MakeAppEnv(o), nil
	case "UiPathTarget":
		var o *targetsv1alpha1.UiPathTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return uipathtarget.MakeAppEnv(o), nil
	case "ZendeskTarget":
		var o *targetsv1alpha1.ZendeskTarget
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return zendesktarget.MakeAppEnv(o), nil
	}
	return nil, fmt.Errorf("kind %q is not supported", object.GetKind())
}
