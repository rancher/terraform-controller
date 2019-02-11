terraform-operator
========

## ***Use K8s to Run Terraform***

Terraform-operator has two components to it, the operator and the executor. 

The operator creates three CRDs and runs controllers for modules and executions. A module is the building block and is the same as a terraform module. This is referenced from an execution which is used to combine all information needed to run Terraform. The execution combines Terraform variables and environment variables from secrets and/or config maps to provide to the executor. 

The executor is a job that runs Terraform. Taking input from the execution run CRD the executor runs `terraform init`, `terraform plan` and `terraform create/destroy` depending on the context.

Executions have a 1-to-many relationship with execution runs, as updates or changes are made in the module or execution additional runs are created to update the terraform resources.

## Building

`make`


## Running

`./bin/terraform-operator`

## Running the Executor in Docker - Useful for testing the Executor
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
