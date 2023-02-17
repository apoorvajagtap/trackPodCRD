Steps to build and run:

1. Start a minikube/kind cluster in local env:

2. Once the cluster is up and running, Clone the repository to create a CRD:
```
$ kubectl apply -f manifests/crd.yaml
$ kubectl api-resources | grep -i trackpod
$ kubectl get crd | grep -i trackpod
```

3. Build and run:
```
$ git clone git@github.com:apoorvajagtap/trackPodCRD.git
$ cd trackPodCRD
$ go build -o main .
$ ./main
```

4. On a different terminal, create a CR, and observe the changes:
(you can modify the count & message as required)
```
$ kubectl apply -f <path_to_trackPodCRD_cloned_repo>/manifests/trackPod.yaml
```

- As soon as the CR is created, new pods with the tpod's name as prefix should be created in current namespace:
```
$ kubectl get pods
$ kubectl get tpod <tpod_name> -oyaml
```

- Keep a watch, and once all the pods are running, the status of CR shall be updated accordingly.
- Modify the CRs spec and observe the further changes.