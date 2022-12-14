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

package broker

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	eventingbroker "github.com/triggermesh/brokers/pkg/config/broker"
	eventingv1alpha1 "github.com/triggermesh/triggermesh-core/pkg/apis/eventing/v1alpha1"

	"github.com/triggermesh/tmctl/pkg/kubernetes"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
)

const (
	dockerHost = "http://host.docker.internal"
)

var _ triggermesh.Component = (*Trigger)(nil)

type Trigger struct {
	Name          string
	ConfigBase    string
	ComponentName string
	LocalURL      *apis.URL

	eventingv1alpha1.TriggerSpec `yaml:"spec,omitempty"`
}

func (t *Trigger) AsK8sObject() (kubernetes.Object, error) {
	spec := map[string]interface{}{
		"broker": t.Broker,
		"target": t.Target,
	}
	if len(t.Filters) != 0 {
		spec["filters"] = t.Filters
	}
	return kubernetes.Object{
		APIVersion: APIVersion,
		Kind:       TriggerKind,
		Metadata: kubernetes.Metadata{
			Name:      t.Name,
			Namespace: triggermesh.Namespace,
			Labels: map[string]string{
				"triggermesh.io/context": t.Broker.Name,
			},
		},
		Spec: spec,
	}, nil
}

// TODO
func (t *Trigger) AsDockerComposeObject() (*triggermesh.DockerComposeService, error) {
	return nil, nil
}

func (t *Trigger) GetKind() string {
	return TriggerKind
}

func (t *Trigger) GetName() string {
	return t.Name
}

func (t *Trigger) GetAPIVersion() string {
	return APIVersion
}

func (t *Trigger) GetSpec() map[string]interface{} {
	return map[string]interface{}{
		"filters": t.Filters,
		"target":  t.Target,
	}
}

func NewTrigger(name, broker, configBase string, target triggermesh.Component, filter *eventingbroker.Filter) (triggermesh.Component, error) {
	trigger := &Trigger{
		Name:       name,
		ConfigBase: configBase,
		TriggerSpec: eventingv1alpha1.TriggerSpec{
			Broker: duckv1.KReference{
				Name:  broker,
				Kind:  BrokerKind,
				Group: "eventing.triggermesh.io",
			},
		},
	}

	if name == "" {
		filterStruct, _ := yaml.Marshal(filter)
		// in case of event types hash collision, replace with sha256
		hash := md5.Sum([]byte(fmt.Sprintf("%s-%s", target.GetName(), string(filterStruct))))
		trigger.Name = fmt.Sprintf("%s-trigger-%s", broker, hex.EncodeToString(hash[:4]))
	}

	if target != nil {
		trigger.ComponentName = target.GetName()
		targetPort, err := target.(triggermesh.Consumer).GetPort(context.Background())
		if err != nil {
			return nil, fmt.Errorf("target local port: %w", err)
		}
		trigger.LocalURL, err = apis.ParseURL(fmt.Sprintf("%s:%s", dockerHost, targetPort))
		if err != nil {
			return nil, fmt.Errorf("target local URL: %w", err)
		}
		trigger.Target = duckv1.Destination{
			Ref: &duckv1.KReference{
				Kind:       target.GetKind(),
				Name:       target.GetName(),
				APIVersion: target.GetAPIVersion(),
			},
		}
	}

	if filter != nil {
		trigger.Filters = []eventingbroker.Filter{*filter}
	}
	return trigger, nil
}

func (t *Trigger) SetTarget(target triggermesh.Component) {
	t.ComponentName = target.GetName()
	t.Target = duckv1.Destination{
		Ref: &duckv1.KReference{
			Kind:       target.GetKind(),
			Name:       target.GetName(),
			APIVersion: target.GetAPIVersion(),
		},
	}
	if consumer, ok := target.(triggermesh.Consumer); ok {
		port, err := consumer.GetPort(context.Background())
		if err != nil {
			return
		}
		t.LocalURL, err = apis.ParseURL(fmt.Sprintf("%s:%s", dockerHost, port))
		if err != nil {
			return
		}
	}
}

func (t *Trigger) LookupTarget() {
	config, err := readBrokerConfig(filepath.Join(t.ConfigBase, t.Broker.Name, brokerConfigFile))
	if err != nil {
		return
	}
	localTrigger, exists := config.Triggers[t.Name]
	if !exists {
		return
	}
	if url, _ := apis.ParseURL(localTrigger.Target.URL); url != nil {
		t.LocalURL = url
	}
	t.ComponentName = localTrigger.Target.Component
	t.Filters = localTrigger.Filters
	t.Target = duckv1.Destination{
		Ref: &duckv1.KReference{
			Name: localTrigger.Target.Component,
		},
	}
}

func FilterExactAttribute(attribute, value string) *eventingbroker.Filter {
	return &eventingbroker.Filter{
		Exact: map[string]string{attribute: strings.TrimSpace(value)},
	}
}
