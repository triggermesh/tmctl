## tmctl send-event

Send CloudEvent to the target

```
tmctl send-event [--eventType <type>][--target <name>] <data> [flags]
```

### Examples

```
tmctl send-event '{"hello":"world"}'
```

### Options

```
      --eventType string   CloudEvent Type attribute (default "triggermesh-local-event")
  -h, --help               help for send-event
      --target string      Component to send the event to. Default is the broker
```

### Options inherited from parent commands

```
      --version string   TriggerMesh components version. (default "v1.25.1")
```

### SEE ALSO

* [tmctl](tmctl.md)	 - A command line interface to build event-driven applications

