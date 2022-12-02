## tmctl create trigger

Create TriggerMesh trigger. More information at https://docs.triggermesh.io/brokers/triggers/

```
tmctl create trigger --target <name> [--source <name>...][--eventTypes <type>...] [flags]
```

### Examples

```
tmctl create trigger --target sockeye --source foo-httppollersource
```

### Options

```
      --eventTypes strings   Event types filter
  -h, --help                 help for trigger
      --name string          Trigger name
      --source strings       Event sources filter
      --target string        Target name
```

### Options inherited from parent commands

```
      --broker string    Optional broker name.
      --version string   TriggerMesh components version.
```

### SEE ALSO

* [tmctl create](tmctl_create.md)	 - Create TriggerMesh component

