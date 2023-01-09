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
	"errors"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights"
	"github.com/Azure/go-autorest/autorest"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
)

const ActivityLogsCrudTimeout = time.Second * 15

func ActivityLogsDiagsName(src *v1alpha1.AzureActivityLogsSource) string {
	// hardcoded value from the upstream
	return "io.triggermesh.azureactivitylogssource." + src.Namespace + "." + src.Name
}

func IsNotFoundErr(err error) bool {
	if dErr := (autorest.DetailedError{}); errors.As(err, &dErr) {
		return dErr.StatusCode == http.StatusNotFound
	}
	return false
}

func ActivityLogsClient(src *v1alpha1.AzureActivityLogsSource, authorizer autorest.Authorizer) (insights.EventCategoriesClient, insights.DiagnosticSettingsClient, error) {
	eventCatCli := insights.NewEventCategoriesClient(src.Spec.SubscriptionID)
	eventCatCli.Authorizer = authorizer

	diagSettingsCli := insights.NewDiagnosticSettingsClient(src.Spec.SubscriptionID)
	diagSettingsCli.Authorizer = authorizer
	return eventCatCli, diagSettingsCli, nil
}
