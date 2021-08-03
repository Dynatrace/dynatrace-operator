# Troubleshoot

Run this script while connected to the affected cluster.

## Instructions

Run the following command to run the script.

```
sh -c "$(curl -fsSL https://raw.githubusercontent.com/dynatrace/dynatrace-operator/master/troubleshoot/troubleshoot.sh)"
```

Make sure to inspect the contents of the troubleshoot script before executing it.

### Manual Instructions

1. Download `troubleshoot.sh` to host with access to affected cluster.
1. Make sure file is executable: `chmod +x troubleshoot.sh`
1. Run script: `./troubleshoot.sh`

## Options

Specify options by appending them to the command, e.g: `./troubleshoot.sh --dynakube dynakube`

`--dynakube <dynakube>`
- allows checking a different dynakube object, by specifying its name
- default: `dynakube`

`--namespace <namespace>`
- allows specifying a different namespace
- default: `dynatrace`

`--oc`
- changes CLI to `oc`
- default: `kubectl`

`--openshift`
- changes the default image to `registry.connect.redhat.com/dynatrace/oneagent`
- default: `docker.io/dynatrace/oneagent`
