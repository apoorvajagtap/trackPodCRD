Steps to build and run:

1. Start a minikube/kind cluster in local env.

2. Once the cluster is up & running, clone the repo in the local environment to build & run.
```
$ git clone git@github.com:apoorvajagtap/trackPodCRD.git
$ cd trackPodCRD
$ make build
$ bin/main
```

3. Now, on a different terminal, create the required resources (CRD & CR). Can opt for creating `setup_trackpod.sh` or `setup_pipelineTask.sh` .
```
$ hack/setup_*.sh . all
```

- As soon as the CR is created, new pods with the relevant CR name's as prefix should be created in current namespace:
```
$ kubectl get pods
$ kubectl get tpod <tpod_name> -oyaml
```

- If running pipelineTask:
```
$ kubectl get prun
$ kubectl get trun
```

- Keep a watch, and once all the pods are running/completed, the status of CR shall be updated accordingly.
- Modify the CRs spec and observe the further changes.