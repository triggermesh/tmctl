## tmctl create transformation

Create TriggerMesh transformation. More information at https://docs.triggermesh.io/transformation/jsontransformation/

```
tmctl create transformation [--target <name>][--source <name>...][--eventTypes <type>...][--from <path>] [flags]
```

### Examples

```
tmctl create transformation <<EOF
  data:
  - operation: add
    paths:
    - key: new-field
      value: hello from Transformation!
EOF
```

### Options

```
      --eventTypes strings   Event types filter
  -f, --from string          Transformation specification file
  -h, --help                 help for transformation
      --name string          Transformation name
      --source strings       Sources component names
      --target string        Target name
```

### Options inherited from parent commands

```
      --version string   TriggerMesh components version. (default "v1.23.3")
```

### SEE ALSO

* [tmctl create](tmctl_create.md)	 - Create TriggerMesh component

