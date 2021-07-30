# Troubleshoot

Run this script while connected to the affected cluster.

## Instructions

1. Download `troubleshoot.sh` to host with access to affected cluster.
1. Make sure file is executable: `chmod +x troubleshoot.sh`
1. Run `./troubleshoot.sh`

## Options

`--dynakube <dynakube>`
- allows checking a different dynakube object, by specifying its name
- default: `dynakube`

`--oc`
- changes used cli to `oc`
- default: `kubectl`

`--openshift`
- changes the default image to `registry.connect.redhat.com/dynatrace/oneagent`
- default: `docker.io/dynatrace/oneagent`
