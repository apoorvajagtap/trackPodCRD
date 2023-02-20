Steps to build and run:

1. Start a minikube/kind cluster in local env.

2. Once the cluster is up & running, clone the repo in the local environment to build & run.
```
$ git clone git@github.com:apoorvajagtap/trackPodCRD.git
$ cd trackPodCRD
$ go build -o main .
$ ./main
```

3. Now, on a different terminal, create the required resources (CRD & CR).
```
$ hack/setup.sh . all
```

- As soon as the CR is created, new pods with the tpod's name as prefix should be created in current namespace:
```
$ kubectl get pods
$ kubectl get tpod <tpod_name> -oyaml
```

- Keep a watch, and once all the pods are running, the status of CR shall be updated accordingly.
- Modify the CRs spec and observe the further changes.