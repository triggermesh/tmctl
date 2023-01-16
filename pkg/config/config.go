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

package config

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	defaultConfigPath = ".triggermesh/cli"
	defaultConfigFile = "config.yaml"
	defaultContext    = ""

	defaultTmVersion     = "v1.23.0"
	defaultBrokerVersion = "v1.1.0"

	MemoryBrokerImage = "gcr.io/triggermesh/memory-broker"
	RedisBrokerImage  = "gcr.io/triggermesh/redis-broker"

	// In-memory broker params
	defaultMemoryBufferSize = "100"
	defaultProduceTimeout   = "1s"
	// Broker config polling period. On Windows only.
	defaultConfigPollingPeriod = "PT2S"
)

type Config struct {
	// Calculated attributes
	ConfigHome string `yaml:"-"`
	CRDPath    string `yaml:"-"`

	// Persisted attributes
	Context     string   `yaml:"context"`
	Triggermesh TmConfig `yaml:"triggermesh"`
}

type TmConfig struct {
	ComponentsVersion string       `yaml:"version"`
	Broker            BrokerConfig `yaml:"broker"`
}

type BrokerConfig struct {
	Image   string                `yaml:"image,omitempty"` // deprecated
	Version string                `yaml:"version"`
	Memory  *InMemoryBrokerConfig `yaml:"memory,omitempty"`
	Redis   *RedisBrokerConfig    `yaml:"redis,omitempty"`
	// for Windows only
	ConfigPollingPeriod string `yaml:"config-polling-period,omitempty"`
}

type InMemoryBrokerConfig struct {
	BufferSize     string `yaml:"buffer-size"`
	ProduceTimeout string `yaml:"produce-timeout"`
}

type RedisBrokerConfig struct {
	Address    string `yaml:"address"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	TLSEnabled bool   `yaml:"tls-enabled,omitempty"`
	SkipVerify bool   `yaml:"skip-verify,omitempty"`
}

func New() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return nil, err
	}
	c := &Config{
		ConfigHome: filepath.Join(absHome, defaultConfigPath),
	}
	if err = c.load(); os.IsNotExist(err) {
		if err := c.createDefault(); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	if err := c.applyOverrides(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) createDefault() error {
	if err := os.MkdirAll(c.ConfigHome, os.ModePerm); err != nil {
		return err
	}
	c.Context = defaultContext
	c.Triggermesh.ComponentsVersion = latestOrDefaultTag("triggermesh", defaultTmVersion)
	c.Triggermesh.Broker.Version = latestOrDefaultTag("brokers", defaultBrokerVersion)
	c.Triggermesh.Broker.Memory = &InMemoryBrokerConfig{
		BufferSize:     defaultMemoryBufferSize,
		ProduceTimeout: defaultProduceTimeout,
	}
	if runtime.GOOS == "windows" {
		c.Triggermesh.Broker.ConfigPollingPeriod = defaultConfigPollingPeriod
	}
	return c.Save()
}

func (c *Config) load() error {
	configFile, err := os.ReadFile(filepath.Join(c.ConfigHome, defaultConfigFile))
	if err != nil {
		return err
	}
	return yaml.Unmarshal(configFile, c)
}

func latestOrDefaultTag(project, defaultVersion string) string {
	r, err := http.Get("https://api.github.com/repos/triggermesh/" + project + "/releases/latest")
	if err != nil {
		return defaultVersion
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return defaultVersion
	}
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&release); err != nil {
		return defaultVersion
	}
	return release.TagName
}

func (c *Config) Save() error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(c.ConfigHome, defaultConfigFile), data, 0644)
}

func (c *Config) Get(key string) string {
	if key == "" {
		out, err := yaml.Marshal(c)
		if err != nil {
			panic(err)
		}
		return string(out)
	}
	return readValue(strings.Split(key, "."), reflect.TypeOf(*c), reflect.ValueOf(*c))
}

func (c *Config) Set(key, value string) error {
	setValue(strings.Split(key, "."), value, reflect.TypeOf(*c), reflect.ValueOf(c))
	return c.Save()
}

func setValue(keys []string, value string, t reflect.Type, v reflect.Value) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Tag.Get("yaml") == keys[0] || field.Tag.Get("yaml") == keys[0]+",omitempty" {
			if len(keys) == 1 {
				switch v.Kind() {
				case reflect.Struct:
					if vv := v.FieldByName(field.Name); vv.Kind() == reflect.String {
						vv.SetString(value)
					}
				case reflect.Pointer:
					if vv := v.Elem().FieldByName(field.Name); vv.Kind() == reflect.String {
						vv.SetString(value)
					}
				}
				return
			}
			switch v.Kind() {
			case reflect.Pointer:
				setValue(keys[1:], value, field.Type, v.Elem().FieldByName(field.Name))
			case reflect.Struct:
				setValue(keys[1:], value, field.Type, v.FieldByName(field.Name))
			}
		}
	}
}

func readValue(keys []string, t reflect.Type, v reflect.Value) string {
	var j int
	var key string
	switch t.Kind() {
	case reflect.Pointer:
		return readValue(keys[j:], t.Elem(), reflect.Indirect(v))
	case reflect.Struct:
		for j, key = range keys {
			for i := 0; i < t.NumField(); i++ {
				field := t.Field(i)
				if field.Tag.Get("yaml") == key || field.Tag.Get("yaml") == key+",omitempty" {
					if !v.IsValid() {
						break
					}
					value := reflect.Indirect(v).FieldByName(field.Name)
					return readValue(keys[j:], field.Type, value)
				}
			}
		}
	case reflect.String:
		if len(keys) == 1 {
			return v.String()
		}
	}
	return ""
}
