# bb-dist-noti-operator
This Kubernetes operator automatically manages node roles (leader/follower) 
for applications running as StatefulSets by monitoring their leader status via gRPC. 
It simplifies the process of identifying and tracking leader and follower nodes in distributed systems 
within Kubernetes.



## Description

The `bb-dist-noti-operator` is specifically designed to work in conjunction with the 
[bb-dist-noti](https://github.com/opplieam/bb-dist-noti/) application (and similar distributed systems) to enhance 
routing and management within Kubernetes.  `bb-dist-noti` and comparable applications often utilize a 
leader-follower architecture. This operator automates the critical task of identifying and labeling leader and follower 
pods within a Kubernetes StatefulSet deployment of such applications.

A key intended use case is to integrate this operator with an Nginx Ingress controller. By leveraging the `node-role` 
labels applied by the `bb-dist-noti-operator`, you can configure your Nginx Ingress to intelligently 
route incoming HTTP traffic **exclusively to the follower nodes** of your application.  
This setup is beneficial in scenarios where follower nodes are designed to handle read requests or 
other types of user traffic, while the leader node is primarily responsible for write operations, coordination, 
or background tasks.

By automatically and dynamically managing `node-role` labels, the `bb-dist-noti-operator` simplifies the configuration 
and ensures the consistent and accurate routing of traffic based on the real-time leader/follower status of 
your distributed application. This leads to improved application availability, scalability, 
and maintainability by decoupling routing logic from manual pod status tracking.

## Getting Started

### Prerequisites
- go version v1.23.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/bb-dist-noti-operator:tag
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
make deploy IMG=<some-registry>/bb-dist-noti-operator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

Set `localDev` to false to prevent gRPC calls during local development.

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following the options to release and provide this solution to the users.

### By providing a bundle with all YAML files

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/bb-dist-noti-operator:tag
```

**NOTE:** The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without its
dependencies.

2. Using the installer

Users can just run 'kubectl apply -f <URL for YAML BUNDLE>' to install
the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/bb-dist-noti-operator/<tag or branch>/dist/install.yaml
```

### By providing a Helm Chart

1. Build the chart using the optional helm plugin

```sh
kubebuilder edit --plugins=helm/v1-alpha
```

2. See that a chart was generated under 'dist/chart', and users
can obtain this solution from there.

**NOTE:** If you change the project, you need to update the Helm Chart
using the same command above to sync the latest changes. Furthermore,
if you create webhooks, you need to use the above command with
the '--force' flag and manually ensure that any custom configuration
previously added to 'dist/chart/values.yaml' or 'dist/chart/manager/manager.yaml'
is manually re-applied afterwards.


## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

