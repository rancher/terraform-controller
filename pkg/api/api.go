package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/rancher/terraform-controller/pkg/api/routes"
	"github.com/rancher/terraform-controller/pkg/file"
	"github.com/sirupsen/logrus"
)

func Start(ctx context.Context, address, certFile, keyFile string) error {
	logrus.Info("Starting API Server")
	r := gin.Default()
	routes.Register(r)

	if certFile != "" && file.Exists(certFile) &&
		keyFile != "" && file.Exists(keyFile) {
		return r.RunTLS(address, certFile, keyFile)
	}

	return r.Run()
}
