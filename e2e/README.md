# End to End Testing

This folder contains all End to End (e2e) testing code 

## Prerequisites

- The e2e tests require you to run `make build-controller` at least once because the binary `./bin/terraform-controller` needs to be available to the e2e tests.
- These tests are expecting a new clean k8s cluster and does not clean up after itself so it is expected that the cluster will be thrown away after e2e tests are complete. You will want to look at k3s/k3d to make this process smoother. Using the `make` scripts are the best way to run these and you are using them at your own risk on an existing cluster.

## Usage

To build/run evertything, use:

```
make
```

To target only the e2e tests, use:

```
make build-controller #created ./bin/terraform-controller
make e2e
```

## Running Tests Locally with k3d

If you'd like to run against a local k3s, it is recommended that you use [k3d](https://github.com/rancher/k3d).

To boot a cluster, run:

```
k3d create --name e2e
```

Then use the provided config to boot the controller. You can use a command like this:

```
KUBECONFIG="$(k3d get-kubeconfig --name='e2e')" ./terraform-controller --threads 1
```

After the controller is running locally, you can run the tests in the same manner:

```
KUBECONFIG="$(k3d get-kubeconfig --name='e2e')" go test -json -count=1 ./e2e/...
```

The `count=1` option is needed because tests will be cached if the code doesn't change, even though you are running the tests against a new cluster with no data. In this case, you would want to re-run the tests for a new cluster.

# Initializing

There is an initilization process which mimics `kubectl create -f ./manifests`. This process assumes that you are running a new k3s server, and tests could fail if it tries to make something that already exists. Therefore, if you are doing local development of the e2e tests, it is recommended to delete and recreate your k3s cluster before performing another run.

To automate the process of deleting your k3s cluster before you recreate it, you can use a Run/Debug configuration in Goland or use a one-line shell tool.

# Terraform Module

We use a [test module](https://github.com/luthermonson/terraform-controller-test-module) to test executions. It uses the k8s Terraform provider and creates a ConfigMap. The e2e tests validate that they were created.

Please note that the e2e test module uses the default k3s setting to boot the API server on https://10.43.0.1 as seen [here](https://github.com/luthermonson/terraform-controller-test-module/blob/master/main.tf#L5). If you change the `--api-port` setting, you will need to change this line to match.
