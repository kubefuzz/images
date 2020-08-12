# AFL Sync

Requires at least Go 1.13 (because Go modules are used).

Can only be executed inside a Kubernetes cluster, since it relies exclusively
on the Kubernetes API.

If you are using CRI-O, make sure to use a current version which includes the fix
[exec: Close pipe fds to prevent hangs](https://github.com/cri-o/cri-o/pull/3243).

## Build

`docker build -t fuzz/afl-sync .`

## Run

Can be run as a Kubernetes CronJob:

```yaml
apiVersion: batch/v1beta1
kind: CronJob
spec:
  schedule: "*/5 * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: afl-sync
          containers:
            - name: sync
              image: kubefuzz/afl-sync:latest
              env:
                - name: SYNC_STATS_ONLY
                  value: "0"  # Set to "1" if only stats should be synced
                - name: POD_NAMESPACE
                  valueFrom:
                    fieldRef:
                      fieldPath: metadata.namespace

```

> The job has to be run with a service-account that has the "pods/exec" permission.
