package routes

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/go-tfe"
	v1 "github.com/rancher/terraform-controller/pkg/apis/terraformcontroller.cattle.io/v1"
	"github.com/rancher/terraform-controller/pkg/types"
)

var cs *types.Controllers

func Register(r *gin.Engine, controllers *types.Controllers) error {
	cs = controllers
	r.GET("/api/v2/ping", ping)
	r.GET("/.well-known/terraform.json", discovery)
	r.GET("/api/v2/organizations/:org/entitlement-set", entitlement)
	r.GET("/api/v2/organizations/:org/workspaces/:workspace", workspace)
	r.GET("/api/v2/workspaces/:workspace/current-state-version", state)
	r.GET("/api/v2/download/:workspace/state", stateDownload)
	r.POST("/api/v2/workspaces/:workspace/actions/lock", stateLock)
	r.POST("/api/v2/workspaces/:workspace/actions/unlock", stateUnlock)
	r.POST("/api/v2/workspaces/:workspace/state-versions", stateUpdate)

	return nil
}

func ping(c *gin.Context) {
	c.String(200, "pong")
}

func discovery(c *gin.Context) {
	c.JSON(200, gin.H{
		"tfe.v2":   "/api/v2/",
		"tfe.v2.1": "/api/v2/",
		"tfe.v2.2": "/api/v2/",
	})
}

func getWorkspace(ws *v1.Workspace) *tfe.Workspace {
	return &tfe.Workspace{
		Name:      ws.ObjectMeta.Name,
		ID:        ws.ObjectMeta.Name,
		AutoApply: ws.Spec.AutoApply,
	}
}

func getStateName(state string) string {
	return fmt.Sprintf("tfstate-default-%s", state)
}
func getLockName(state string) string {
	return fmt.Sprintf("lock-tfstate-default-%s", state)
}

func badRequest(c *gin.Context, err error) {
	logrus.Debug(err)
	c.JSON(http.StatusBadRequest, gin.H{
		"error": err.Error(),
	})
}

func notFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{})
}

func gunzip(data []byte) (string, error) {
	var r io.Reader
	var err error

	b := bytes.NewBuffer(data)
	r, err = gzip.NewReader(b)
	if err != nil {
		return "", err
	}

	var resB bytes.Buffer
	_, err = resB.ReadFrom(r)
	if err != nil {
		return "", err
	}

	return string(resB.Bytes()), nil
}

func gzipData(data []byte) (compressedData []byte, err error) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)

	_, err = gz.Write(data)
	if err != nil {
		return
	}

	if err = gz.Flush(); err != nil {
		return
	}

	if err = gz.Close(); err != nil {
		return
	}

	compressedData = b.Bytes()

	return
}
