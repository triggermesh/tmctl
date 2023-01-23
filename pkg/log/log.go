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

package log

import (
	glog "log"

	"github.com/triggermesh/tmctl/pkg/config"
)

var broker = "-"

func init() {
	c, _ := config.New()
	if c != nil && c.Context != "" {
		broker = c.Context
	}
}

// Println is standard's log output supplied with the broker name.
func Println(message string) {
	glog.Printf("%s | %s", broker, message)
}

// Printf is standard's log formattable output supplied with the broker name.
func Printf(format string, v ...any) {
	glog.Printf(broker+" | "+format, v...)
}

// Fatal is the local fatal function.
func Fatal(v ...any) {
	glog.Fatal(v...)
}
