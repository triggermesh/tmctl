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
	"fmt"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	autorestauth "github.com/Azure/go-autorest/autorest/azure/auth"
)

func Client(secrets map[string]string) (autorest.Authorizer, error) {
	tenantID, exists := secrets["tenantID"]
	if !exists {
		return nil, fmt.Errorf("\"tenantID\" spec value is missing")
	}
	clientID, exists := secrets["clientID"]
	if !exists {
		return nil, fmt.Errorf("\"clientID\" spec value is missing")
	}
	clientSecret, exists := secrets["clientSecret"]
	if !exists {
		return nil, fmt.Errorf("\"clientSecret\" spec value is missing")
	}
	azureEnv := &azure.PublicCloud
	authSettings := autorestauth.EnvironmentSettings{
		Values: map[string]string{
			autorestauth.TenantID:     tenantID,
			autorestauth.ClientID:     clientID,
			autorestauth.ClientSecret: clientSecret,
			autorestauth.Resource:     azureEnv.ResourceManagerEndpoint,
		},
		Environment: *azureEnv,
	}
	return authSettings.GetAuthorizer()
}
