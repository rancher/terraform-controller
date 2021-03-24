package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/google/jsonapi"
	"github.com/hashicorp/go-tfe"
	"github.com/rancher/terraform-controller/pkg/types"
)

var cs *types.Controllers

func Register(r *gin.Engine, controllers *types.Controllers) error {
	cs = controllers
	r.GET("/api/v2/ping", ping)
	r.GET("/.well-known/terraform.json", discovery)
	r.GET("/api/v2/organizations/:org/entitlement-set", entitlement)
	r.GET("/api/v2/organizations/:org/workspaces/:workspace", workspace)

	return nil
}

func ping(c *gin.Context) {
	c.String(200, "pong")
}

func entitlement(c *gin.Context) {
	c.Header("Content-Type", jsonapi.MediaType)
	ent := &tfe.Entitlements{
		Operations: true,
	}
	jsonapi.MarshalPayload(c.Writer, ent)
}

func discovery(c *gin.Context) {
	c.JSON(200, gin.H{
		"tfe.v2":   "/api/v2/",
		"tfe.v2.1": "/api/v2/",
		"tfe.v2.2": "/api/v2/",
	})
}

func workspace(c *gin.Context) {
	workspace := &tfe.Workspace{}
	workspace.Name = c.Param("workspace")
	workspace.ID = "ws-123"
	c.Header("Content-Type", jsonapi.MediaType)
	jsonapi.MarshalPayload(c.Writer, workspace)
}
