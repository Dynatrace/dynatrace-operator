# Rules for debug logging

- Do not log errors that bubble up to the controller runtime. They will be logged anyways and we do not want to log errors multiple times because it's confusing.
- Use debug logs to show the flow.
- Provide meta data so that logs can be related to objects under reconciliation
- Provide additional information that might be helpful for troubleshoot in key/values of log (e.g. values of variables at that point)
- Use a local logger with pre-configured key/values to avoid duplication
- Be careful to not accidentally log confidential info like passwords or tokens.


## Examples for debug logs

### Show the flow

```go
log.Debug("reconcile required", "updater", updater.Name())
```

### Something's wrong

```go
if err != nil {
    log.Debug("could not create or update deployment for EdgeConnect", "name", desiredDeployment.Name)

    return err
}

log.Debug("EdgeConnect deployment created/updated", "name", edgeConnect.Name)
```

### Logging additional info

```go
if len(ecs.EdgeConnects) > 1 {
    log.Debug("Found multiple EdgeConnect objects with the same name", "count", ecs.EdgeConnects)

    return edgeconnect.GetResponse{}, errors.New("many EdgeConnects have the same name")
}

```


### Pre-configured local logger

```go
func (controller *Controller) reconcileEdgeConnectDeletion(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect) error {
    llog := log.WithValues("namespace", edgeConnect.Namespace, "name", edgeConnect.Name)

    ...
	llog.Debug("foobar happened")

	llog = llog.WithValues("foobarReaseon", "hurga")

	...
	llog.Debug("foobar happened again")
    ...
}

```
