## tmctl create source

Create TriggerMesh source. More information at https://docs.triggermesh.io

```
tmctl create source [kind]/[--from-image <image>][--name <name>] [flags]
```

### Examples

```
tmctl create source httppoller \
	--endpoint https://www.example.com \
	--eventType sample-event \
	--interval 30s  \
	--method GET
```

### Options

```
  -h, --help   help for source
```

### Options inherited from parent commands

```
      --version string   TriggerMesh components version. (default "v1.23.1")
```

### SEE ALSO

* [tmctl create](tmctl_create.md)	 - Create TriggerMesh component

