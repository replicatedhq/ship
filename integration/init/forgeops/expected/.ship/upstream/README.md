# ForgeRock Directory Services Helm chart

Deploy one or more ForgeRock Directory Server instances using Persistent disk claims
and StatefulSets.

## Sample Usage

To deploy to a Kubernetes cluster:

`helm install --set "instance=userstore" ds`

This will install a sample DS userstore.

The instance will be available in the cluster as userstore-0.

If you wish to connect an ldap browser on your local machine to this instance, you can use:

`kubectl port-forward userstore-0 1389:1389`

And open up a connection to ldap://localhost:1389

The default password is "password".

## Persistent Disks

The statefulset uses a Persistent volume claim template to allocate storage for each directory server pod. Persistent volume claims are not deleted when the statefulset is deleted.  In other words, performing a `helm delete ds-release`  will *not* delete the underlying storage. If you want to reclaim the storage, delete the PVC:

```bash
kubectl get pvc
kubectl delete pvc userstore-0
```

## Values.yaml

Please refer to values.yaml. There are a number of variables you can set on the helm command line, or
in your own custom.yaml to control the behavior of the deployment. The features described below
are all controlled by variables in values.yaml.

## Diagnostics and Troubleshooting

Use kubectl exec to get a shell into the running container. For example:

`kubectl exec userstore-0 -it bash`

There are a number of utility scripts found under `/opt/opendj/scripts`, as well as the 
directory server commands in `/opt/opendj/bin`.

use kubectl logs to see the pod logs. 

`kubectl logs userstore-0 -f`

## Scaling and replication

To scale a deployment set the number of replicas in values.yaml. See values.yaml
for the various options. Each node in the statefulset is a combined directory and replication server. Note that the topology of the set can not be changed after installation by scaling the statefulset. You can not add or remove ds nodes without reinitializing the cluster from scratch or from a backup. The desired number of ds/rs instances should be planned in advance.


## Backup

If backup is enabled, each pod in the statefulset mounts a shared backup
 volume claim (PVC) on bak/. This PVC holds the contents of the backups. You must size this PVC according 
to the amount of backup data you wish to retain. Old backups must be purged manually. The backup pvc must
be an ReadWriteMany volume type (like NFS, for example). 

A backup can be initiated manually by execing into the image and running the scripts/backup.sh command. For example:

`kubectl exec userstore-0 -it bash`
`./scripts/backup.sh`

The backups can be listed using `scripts/list-backup.sh`

## Restore 

The chart can restore the state of the directory from a previous backup. Set the value restore.enabled=true on deployment.  The restore process will not overwrite a data/ pvc that contains data. 

## Benchmarking 

If you are benchmarking on a cloud provider make sure you use an SSD storage class as the directory is very sensitive to disk performance.
