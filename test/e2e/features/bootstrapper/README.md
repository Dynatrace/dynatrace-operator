# Bootstrapper E2E Tests

## PGC (Process Grouping Config) Test

### What it tests
- Bootstrapper's Process Grouping Config (PGC) feature (commit 06017fd10)
- DT API PGC fetch → source secret → per-namespace secret replication → pod volume projection
- End-to-end flow: verify PGC data flows from Dynatrace API into bootstrapper config

### Test flow
1. Deploy DynaKube with ApplicationMonitoring + node image pull enabled
2. Wait for bootstrapper jobs to complete and clean up
3. Assert source secret `<dk-name>-bootstrapper-config` in operator namespace contains:
   - Non-empty `declarative.cbor` data (PGC CBOR file from DT API)
   - Non-empty `internal.operator.dynatrace.com/pgc-etag` annotation
4. Deploy sample app in separate namespace
5. Assert per-namespace secret `dynatrace-bootstrapper-config` contains same `declarative.cbor`
6. Assert pod's `dynatrace-input` projected volume references `dynatrace-bootstrapper-config`

### Why no init-container exec
The bootstrapper init container exits immediately after pod startup. By assertion time, the container is `Completed` and cannot be exec'd. Secret-level checks + volume projection assertions fully verify the PGC flow.

### Run
```bash
go test ./test/e2e/scenarios/standard -v -tags e2e -run TestStandard_bootstrapper_pgc
```

### Prerequisites
- Test Dynatrace environment must have PGC data configured for the cluster
- Bootstrapper image must be available (env var `E2E_CODEMODULES_IMAGE` or `ghcr.io/dynatrace/dynatrace-bootstrapper:snapshot`)
