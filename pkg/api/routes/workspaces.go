package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/google/jsonapi"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func workspace(c *gin.Context) {
	c.Header("Content-Type", jsonapi.MediaType)
	wsParam := c.Param("workspace")
	ws, err := cs.Workspace.Get("default", wsParam, metav1.GetOptions{})
	if err != nil {
		notFound(c)
		return
	}

	workspace := getWorkspace(ws)
	err = jsonapi.MarshalPayload(c.Writer, workspace)
	if err != nil {
		logrus.Errorf("error marshalling workspace: %s", err)
	}
}
