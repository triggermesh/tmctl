## tmctl delete

Delete components by names

```
tmctl delete <component_name_1, component_name_2...> [--broker <name>] [flags]
```

### Examples

```
tmctl delete foo-httptarget, foo-awss3source
tmctl delete --broker foo
```

### Options

```
      --broker string   Delete the broker
  -h, --help            help for delete
```

### Options inherited from parent commands

```
      --version string   TriggerMesh components version. (default "v1.23.3")
```

### SEE ALSO

* [tmctl](tmctl.md)	 - A command line interface to build event-driven applications

