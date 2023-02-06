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

package test

import (
	"os"
	"path"
	"runtime"

	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

func Manifest() string {
	_, filename, _, _ := runtime.Caller(0)
	return path.Dir(filename) + "/fixtures/manifest.yaml"
}

func CRD() map[string]crd.CRD {
	_, filename, _, _ := runtime.Caller(0)
	reader, err := os.Open(path.Dir(filename) + "/fixtures/crd.yaml")
	if err != nil {
		panic(err)
	}
	crds, err := crd.Parse(reader)
	if err != nil {
		panic(err)
	}
	return crds
}

func ConfigBase() string {
	_, filename, _, _ := runtime.Caller(0)
	return path.Dir(filename) + "/fixtures"
}
