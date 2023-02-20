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

package monitoring

const dashboard string = `
{
    "annotations": {
      "list": [
        {
          "builtIn": 1,
          "datasource": {
            "type": "grafana",
            "uid": "-- Grafana --"
          },
          "enable": true,
          "hide": true,
          "iconColor": "rgba(0, 211, 255, 1)",
          "name": "Annotations & Alerts",
          "target": {
            "limit": 100,
            "matchAny": false,
            "tags": [],
            "type": "dashboard"
          },
          "type": "dashboard"
        }
      ]
    },
    "editable": true,
    "fiscalYearStartMonth": 0,
    "graphTooltip": 0,
    "id": 1,
    "links": [],
    "liveNow": false,
    "panels": [
      {
        "datasource": {
          "type": "prometheus",
          "uid": "PBFA97CFB590B2093"
        },
        "fieldConfig": {
          "defaults": {
            "color": {
              "mode": "palette-classic"
            },
            "custom": {
              "axisCenteredZero": false,
              "axisColorMode": "text",
              "axisLabel": "",
              "axisPlacement": "auto",
              "barAlignment": 0,
              "drawStyle": "line",
              "fillOpacity": 0,
              "gradientMode": "none",
              "hideFrom": {
                "legend": false,
                "tooltip": false,
                "viz": false
              },
              "lineInterpolation": "linear",
              "lineWidth": 1,
              "pointSize": 5,
              "scaleDistribution": {
                "type": "linear"
              },
              "showPoints": "auto",
              "spanNulls": false,
              "stacking": {
                "group": "A",
                "mode": "none"
              },
              "thresholdsStyle": {
                "mode": "off"
              }
            },
            "mappings": [],
            "thresholds": {
              "mode": "absolute",
              "steps": [
                {
                  "color": "green",
                  "value": null
                },
                {
                  "color": "red",
                  "value": 80
                }
              ]
            }
          },
          "overrides": []
        },
        "gridPos": {
          "h": 9,
          "w": 12,
          "x": 0,
          "y": 0
        },
        "id": 2,
        "options": {
          "legend": {
            "calcs": [],
            "displayMode": "list",
            "placement": "bottom",
            "showLegend": true
          },
          "tooltip": {
            "mode": "single",
            "sort": "none"
          }
        },
        "targets": [
          {
            "datasource": {
              "type": "prometheus",
              "uid": "PBFA97CFB590B2093"
            },
            "editorMode": "builder",
            "expr": "broker_ingest_event_count{context=\"$context\"}",
            "legendFormat": "{{received_type}}",
            "range": true,
            "refId": "A"
          }
        ],
        "title": "Broker events",
        "type": "timeseries"
      }
    ],
    "schemaVersion": 37,
    "style": "dark",
    "tags": [],
    "templating": {
      "list": [
        {
          "current": {
            "selected": true,
            "text": "foo",
            "value": "foo"
          },
          "datasource": {
            "type": "prometheus",
            "uid": "PBFA97CFB590B2093"
          },
          "definition": "label_values(context)",
          "hide": 0,
          "includeAll": false,
          "multi": false,
          "name": "context",
          "options": [],
          "query": {
            "query": "label_values(context)",
            "refId": "StandardVariableQuery"
          },
          "refresh": 1,
          "regex": "",
          "skipUrlSync": false,
          "sort": 0,
          "type": "query"
        }
      ]
    },
    "time": {
      "from": "now-5m",
      "to": "now"
    },
    "timepicker": {},
    "timezone": "",
    "title": "Test dashboard",
    "uid": "ckzMKoJ4z",
    "version": 3,
    "weekStart": ""
  }
`

type DataSourceProvision struct {
	APIVersion  int          `yaml:"apiVersion"`
	Datasources []DataSource `yaml:"datasources"`
}

type DataSource struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Access   string `yaml:"access"`
	URL      string `yaml:"url"`
	JSONData struct {
		HTTPMethod                  string `yaml:"httpMethod"`
		ExemplarTraceIDDestinations []struct {
			DatasourceUID string `yaml:"datasourceUid,omitempty"`
			Name          string `yaml:"name"`
			URL           string `yaml:"url,omitempty"`
		} `yaml:"exemplarTraceIdDestinations"`
	} `yaml:"jsonData,omitempty"`
}

func createDataSourceProvision(url string) *DataSourceProvision {
	return &DataSourceProvision{
		APIVersion: 1,
		Datasources: []DataSource{
			{
				Name:   "Prometheus",
				Type:   "prometheus",
				Access: "proxy",
				URL:    url,
			},
		},
	}
}

func createDashboardProvision() []byte {
	return []byte(`apiVersion: 1
providers:
- name: dashboards
  type: file
  updateIntervalSeconds: 30
  options:
    path: /etc/grafana/provisioning/dashboards
    foldersFromFilesStructure: true`)
}
