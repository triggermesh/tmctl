## tmctl create target

Create TriggerMesh target. More information at https://docs.triggermesh.io

```
tmctl create target [kind]/[--from-image <image>][--name <name>][--source <name>...][--eventTypes <type>...] [flags]
```

### Examples

```
tmctl create target http \
	--endpoint https://image-charts.com \
	--method GET \
	--response.eventType qr-data.response
```

### Options

```
  -h, --help   help for target
```

### Options inherited from parent commands

```
      --version string   TriggerMesh components version. (default "v1.23.0")
```

### SEE ALSO

* [tmctl create](tmctl_create.md)	 - Create TriggerMesh component

