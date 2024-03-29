# Angiplatform
Welcome to Angiplatform! This project showcases proficiency in Kubernetes and the reconciliation controller pattern. The primary aim is to develop a controller that manages a custom resource for deploying an application and its associated data store within Kubernetes. The custom resource encapsulates the configuration required for deploying the application.


## Project Overview
In this project, we leverage the podinfo application, an open-source application, and Redis from DockerHub. These components are orchestrated within Kubernetes, with the application and Redis instances accessible via kube port forwarding.


## Getting Started

Follow the instructions below to set up and run the Angiplatform project locally:

### Prerequisites
- go version v1.21.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### Frameworks/Tools used:
- Kubebuilder
- Redis CLI
- Minikube
- Docker
- GO
- Kubernetes-Kubectl
 

### Clone the repo 

```sh 
git clone https://github.com/kommineni24/Angiplatform.git
```

### To Deploy on the cluster

**Build and push your image to the location specified by `IMG`:**

Naviagate to the directory you cloned above and build/tag/push the docker image as below.

```sh
make docker-build docker-push IMG=<some-registry>/angiplatform:tag
```
**NOTE:** This image ought to be published in the personal registry you specified. 
And it is required to have access to pull the image from the working environment. 
Make sure you have the proper permission to the registry if the above commands don’t work.



**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/angiplatform:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin 
privileges or be logged in as admin.


**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

The Custom Resource file that I have used:
```sh
apiVersion: my.api.group.rama.angi.platform/v1alpha1
kind: MyAppResource
metadata:
  labels:
    app.kubernetes.io/name: myappresource
    app.kubernetes.io/instance: myappresource-sample
    app.kubernetes.io/part-of: angiplatform
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/created-by: angiplatform
  name: myappresource-sample
spec:
  # TODO(user): Add fields here
  replicaCount: 2
  resources:
    memoryLimit: 64Mi
    cpuRequest: 100m
  image:
    repository: ghcr.io/stefanprodan/podinfo
    tag: latest
  ui:
    color: "34577c"
    message: "Hey there"
  redis:
    enabled: true
```
 
>**NOTE**: Ensure that the samples has default values to test it out.


### Application verification:

We will rely on kube port forwarding to access the podinfo and redis application.


**Port forwarding for `Podinfo Application and Redis instance`:**

Navigate to the namespace where your application, custom controller, and Redis are deployed.

**Port-forwarding command for application:**
```sh
kubectl port-forward pod/<application-pod-name> <local-port>:<container-port> -n <namespace>
```

>**NOTE**: Replace <application-pod-name> with the name of your application pod, <local-port> with the local port you want to use, <container-port> with the port where your application is running inside the container, and <namespace> with the namespace where your application pod is deployed.

For example:
```sh
kubectl port-forward pod/myappresource-sample-0 8080:9898 -n angiplatform-system
```


Browser Verification URL of application:
```sh
http://localhost:<local-port>
```

**You should see a web page like below:**
![Website shall look like below:](https://github.com/kommineni24/Angiplatform/blob/master/Images/Angi%20Podinfo.png?raw=true)



**Port-forwarding command for Redis:**
```sh
kubectl port-forward pod/<Redis-pod-name> <local-port>:<container-port> -n <namespace>
```

>**NOTE**: Replace <Redis-pod-name> with the name of your Redis pod, <local-port> with the local port you want to use for port-forwarding, and <container-port> with the port where Redis is running inside the container. Also, replace <namespace> with the namespace where your Redis pod is deployed.

For example:
```sh
kubectl port-forward pod/myappresource-sample-redis-6865dcff76-h8r7k 8081:6379 -n angiplatform-system
```


Verification of Redis:
Set key-value pair in Redis:
```sh
redis-cli -h 127.0.0.1 -p <local-port> set platform "Angi"
```

Retrieve value from Redis:
```sh
redis-cli -h 127.0.0.1 -p <local-port> get platform
```

**You can PUT/GET from Redis CLI like below:**
![Redis Verification](https://github.com/kommineni24/Angiplatform/blob/master/Images/Redis%20Verify.png?raw=true)




### Tests:

Naviagte to the folder where our test suite files are and run:
```sh
gingko
```

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/ -n <namespace>
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```


## License

Copyright 2024 Ramakrishna Kommineni.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

