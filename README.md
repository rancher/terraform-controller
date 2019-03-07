[EXPERIMENTAL] terraform-operator
========

## ***Use K8s to Run Terraform***

**NOTE:** We are actively experimenting with this in the open. Consider this ALPHA software and subject to change.

Terraform-operator - This is a low level tool to run Git controlled Terraform modules in Kubernetes. The operator manages the TF state file using Kubernetes as a remote statefile backend! [Backend upstream PR](https://github.com/hashicorp/terraform/pull/19525) You can have changes auto-applied or wait for an explicit "OK" before running. 

There are two parts to the stack, the operator and the executor. 

The operator creates three CRDs and runs controllers for modules and executions. A module is the building block and is the same as a terraform module. This is referenced from an execution which is used to combine all information needed to run Terraform. The execution combines Terraform variables and environment variables from secrets and/or config maps to provide to the executor. 

The executor is a job that runs Terraform. Taking input from the execution run CRD the executor runs `terraform init`, `terraform plan` and `terraform create/destroy` depending on the context.

Executions have a 1-to-many relationship with execution runs, as updates or changes are made in the module or execution additional runs are created to update the terraform resources.

## Quickstart 

Run the testing/development [k3s](https://k3s.io) based TF Operator Appliance Container. 
```
docker run --privileged -d -v $(pwd):/output -e K3S_KUBECONFIG_OUTPUT=/output/kubeconfig.yml -e K3S_KUBECONFIG_MODE=666 -p 6443:6443 rancher/terraform-operator-appliance:v0.0.3 server
```

This will output a kubeconfig.yml file in your local directory. You should edit the file and set the `server:` url to the correct host. If you are using Docker for Mac or Linux the default localhost and port 
ought to work. 

`export KUBECONFIG=$(pwd)/kubeconfig.yml`

# Verify
```
kubectl get all -n tf-operator

NAME                                     READY     STATUS    RESTARTS   AGE
pod/terraform-operator-86f698977-f5nnd   1/1       Running   0          49m

NAME                                 READY     UP-TO-DATE   AVAILABLE   AGE
deployment.apps/terraform-operator   1/1       1            1           49m

NAME                                           DESIRED   CURRENT   READY     AGE
replicaset.apps/terraform-operator-86f698977   1         1         1         49m
```

You now have the operator running and can follow the example in the [folder](https://github.com/rancher/terraform-operator/tree/master/example).

## Building Custom Execution Environment

Create a Dockerfile

```
FROM rancher/terraform-operator-executor:v0.0.3 #Or whatever the release is
RUN curl https://myurl.com/get-some-binary
```

Build that image and push to a registry.

When creating the execution define the image:
```
apiVersion: terraform-operator.cattle.io/v1
kind: Execution
metadata:
  name: cluster-create
spec:
  moduleName: cluster-modules
  destroyOnDelete: true
  autoConfirm: false
  image: cloudnautique/tf-executor-rancher2-provider:v0.0.3 # Custom IMAGE
  variables:
    SecretNames:
    - my-secret
    envConfigNames:
    - env-config
```

If you already have an execution, edit the CR via kubectl and add the image field.

## Building

`make`


### Local Execution

`./bin/terraform-operator`

### Running the Executor in Docker - Useful for testing the Executor
docker run -d -v "/Path/To/Kubeconfig:/root/.kube/config" -e "KUBECONFIG=/root/.kube/config" -e "EXECUTOR_RUN_NAME=RUN_NAME" -e "EXECUTOR_ACTION=create" rancher/terraform-executor:dev

## License
Copyright (c) 2019 [Rancher Labs, Inc.](http://rancher.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
