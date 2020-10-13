module github.com/rancher/terraform-controller

go 1.13

require (
	github.com/docker/go-units v0.4.0
	github.com/go-delve/delve v1.5.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/rancher/lasso v0.0.0-20200905045615-7fcb07d6a20b
	github.com/rancher/wrangler v0.7.2
	github.com/rancher/wrangler-api v0.6.1-0.20200515193802-dcf70881b087
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.4.0
	github.com/urfave/cli v1.20.0
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	k8s.io/api v0.18.8
	k8s.io/apiextensions-apiserver v0.18.0
	k8s.io/apimachinery v0.18.8
	k8s.io/client-go v0.18.8
	k8s.io/gengo v0.0.0-20200114144118-36b2048a9120
	sigs.k8s.io/controller-runtime v0.4.0 // indirect
)

replace github.com/matryer/moq => github.com/rancher/moq v0.0.0-20190404221404-ee5226d43009
