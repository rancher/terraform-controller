module github.com/rancher/terraform-controller

go 1.12

require (
	github.com/pkg/errors v0.8.1
	github.com/rancher/wrangler v0.1.0
	github.com/rancher/wrangler-api v0.1.1
	github.com/sirupsen/logrus v1.4.1
	github.com/stretchr/testify v1.3.0
	github.com/urfave/cli v1.20.0
	k8s.io/api v0.0.0-20190409021203-6e4e0e4f393b
	k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
)

replace github.com/matryer/moq => github.com/rancher/moq v0.0.0-20190404221404-ee5226d43009
