package routes

import (
	"bytes"
	"compress/gzip"
	b64 "encoding/base64"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/google/jsonapi"
	"github.com/hashicorp/go-tfe"
	v1 "github.com/rancher/terraform-controller/pkg/apis/terraformcontroller.cattle.io/v1"
	"github.com/rancher/terraform-controller/pkg/types"
	"github.com/sirupsen/logrus"
	coordv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
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
func workspace(c *gin.Context) {
	c.Header("Content-Type", jsonapi.MediaType)
	wsParam := c.Param("workspace")
	ws, err := cs.Workspace.Get("default", wsParam, metav1.GetOptions{})
	if err != nil {
		logrus.Error(err)
	}
	workspace := getWorkspace(ws)
	jsonapi.MarshalPayload(c.Writer, workspace)
}

func stateLock(c *gin.Context) {
	wsParam := c.Param("workspace")
	lockID := "fake-tfe"
	ws, err := cs.Workspace.Get("default", wsParam, metav1.GetOptions{})
	if err != nil {
		logrus.Error(err)
	}
	workspace := getWorkspace(ws)
	workspace.Locked = true
	lease, _ := cs.Coordination.Get("default", getLockName(ws.Spec.State), metav1.GetOptions{})
	lease.Spec = coordv1.LeaseSpec{HolderIdentity: pointer.StringPtr(lockID)}
	cs.Coordination.Update(lease)
	jsonapi.MarshalPayload(c.Writer, workspace)
}
func stateUnlock(c *gin.Context) {
	wsParam := c.Param("workspace")
	ws, err := cs.Workspace.Get("default", wsParam, metav1.GetOptions{})
	if err != nil {
		logrus.Error(err)
	}
	workspace := getWorkspace(ws)
	workspace.Locked = false
	lease, _ := cs.Coordination.Get("default", getLockName(ws.Spec.State), metav1.GetOptions{})
	lease.Spec.HolderIdentity = nil
	cs.Coordination.Update(lease)
	jsonapi.MarshalPayload(c.Writer, workspace)
}
func state(c *gin.Context) {
	c.Header("Content-Type", jsonapi.MediaType)
	ws := c.Param("workspace")

	stateVersion := &tfe.StateVersion{}
	stateVersion.DownloadURL = fmt.Sprintf("download/%s/state", ws)
	jsonapi.MarshalPayload(c.Writer, stateVersion)
}

func stateUpdate(c *gin.Context) {
	wsParam := c.Param("workspace")
	newState := new(tfe.StateVersionCreateOptions)
	ws, err := cs.Workspace.Get("default", wsParam, metav1.GetOptions{})
	if err != nil {
		logrus.Error(err)
	}

	err = jsonapi.UnmarshalPayload(c.Request.Body, newState)
	if err != nil {
		logrus.Error(err.Error())
	}
	secret, _ := cs.Secret.Get("default", getStateName(ws.Spec.State), metav1.GetOptions{})
	secretData, _ := b64.StdEncoding.DecodeString(*newState.State)
	gzippedData, _ := gzipData(secretData)
	secret.Data["tfstate"] = gzippedData
	cs.Secret.Update(secret)
	stateVersion := tfe.StateVersion{}
	jsonapi.MarshalPayload(c.Writer, stateVersion)

}

func stateDownload(c *gin.Context) {
	c.Header("Content-Type", jsonapi.MediaType)
	wsParam := c.Param("workspace")
	ws, err := cs.Workspace.Get("default", wsParam, metav1.GetOptions{})
	if err != nil {
		logrus.Error(err)
	}
	secret, _ := cs.Secret.Get("default", getStateName(ws.Spec.State), metav1.GetOptions{})
	state, _ := gunzip(secret.Data["tfstate"])
	c.String(200, state)
}

func gunzip(data []byte) (string, error) {
	b := bytes.NewBuffer(data)
	var r io.Reader
	r, err := gzip.NewReader(b)
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
