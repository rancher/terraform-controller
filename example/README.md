## Example

Create the module:

`kubectl create -f module.yaml`

Create the resources the Execution will require to run:

`kubectl create -f secret.yaml -f envvar.yaml`

This creates a secret with the Digital Ocean token for creating the droplet and a config map that will be pulled into the environment of the Execution Run. 

Create the exectution:

`kubectl create -f execution.yaml`

Check the logs of the Execution Run to verify Terraform is going to perform the expected operations:

`kubectl logs [pod-name]`

Assuming the action Terraform is going to perform is correct annotate the Execution Run to approve the changes:

`kubectl annotate executionruns.terraform-operator.cattle.io [execution-run-name] approved="yes" --overwrite`

Once the job completes, you can see the outputs from Terraform by checking the Execution Run:

`kubeclt get executionruns.terraform-operator.cattle.io [execution-run-name] -o yaml`

To remove the resource Terraform created, delete the Execution and follow the same steps to valide the logs from the Job and annotate the Execution Run.