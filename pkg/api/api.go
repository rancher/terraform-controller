package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/rancher/terraform-controller/pkg/api/routes"
	"github.com/sirupsen/logrus"
)

func Start(ctx context.Context, address string) error {
	logrus.Info("Starting API Server")
	r := gin.Default()
	routes.Register(r)

	return r.Run(address)
}
