package routes

import (
	b64 "encoding/base64"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/jsonapi"
	"github.com/hashicorp/go-tfe"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func stateDownload(c *gin.Context) {
	c.Header("Content-Type", jsonapi.MediaType)
	wsParam := c.Param("workspace")

	ws, err := cs.Workspace.Get("default", wsParam, metav1.GetOptions{})
	if err != nil {
		notFound(c)
		return
	}

	var secret *corev1.Secret
	secret, err = cs.Secret.Get("default", getStateName(ws.Spec.State), metav1.GetOptions{})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			badRequest(c, fmt.Errorf("error pulling secret %s: %s", getStateName(ws.Spec.State), err))
			return
		}
		secret.Data = make(map[string][]byte)
	}

	var state string
	if len(secret.Data) > 0 {
		state, err = gunzip(secret.Data["tfstate"])
		if err != nil {
			badRequest(c, fmt.Errorf("error un-gzipping state: %s", err))
			return
		}
	}

	c.String(200, state)
}

func stateLock(c *gin.Context) {
	wsParam := c.Param("workspace")
	lockID := "fake-tfe"
	ws, err := cs.Workspace.Get("default", wsParam, metav1.GetOptions{})
	if err != nil {
		notFound(c)
		return
	}

	workspace := getWorkspace(ws)
	workspace.Locked = true
	lease, err := cs.Coordination.Get("default", getLockName(ws.Spec.State), metav1.GetOptions{})
	if err != nil {
		badRequest(c, fmt.Errorf("error getting coordination %s: %s", getStateName(ws.Spec.State), err))
		return
	}

	lease.Spec = v1.LeaseSpec{
		HolderIdentity: pointer.StringPtr(lockID),
	}

	_, err = cs.Coordination.Update(lease)
	if err != nil {
		badRequest(c, fmt.Errorf("error updating coordination %s: %s", lease.Name, err))
		return
	}

	err = jsonapi.MarshalPayload(c.Writer, workspace)
	if err != nil {
		logrus.Errorf("error marshalling state lock payload: %s", err)
	}
}

func stateUnlock(c *gin.Context) {
	wsParam := c.Param("workspace")
	ws, err := cs.Workspace.Get("default", wsParam, metav1.GetOptions{})
	if err != nil {
		notFound(c)
		return
	}

	workspace := getWorkspace(ws)
	workspace.Locked = false
	lease, err := cs.Coordination.Get("default", getLockName(ws.Spec.State), metav1.GetOptions{})
	if err != nil {
		badRequest(c, fmt.Errorf("error getting coordination %s: %s", getLockName(ws.Spec.State), err))
		return
	}

	lease.Spec.HolderIdentity = nil
	_, err = cs.Coordination.Update(lease)
	if err != nil {
		badRequest(c, fmt.Errorf("error updating coordination %s: %s", lease.Name, err))
		return
	}

	err = jsonapi.MarshalPayload(c.Writer, workspace)
	if err != nil {
		logrus.Errorf("error marshalling state lock payload: %s", err)
	}
}

func state(c *gin.Context) {
	c.Header("Content-Type", jsonapi.MediaType)
	ws := c.Param("workspace")

	stateVersion := &tfe.StateVersion{
		DownloadURL: fmt.Sprintf("download/%s/state", ws),
	}

	err := jsonapi.MarshalPayload(c.Writer, stateVersion)
	if err != nil {
		logrus.Errorf("error marshalling state lock payload: %s", err)
	}
}

func stateUpdate(c *gin.Context) {
	c.Header("Content-Type", jsonapi.MediaType)
	wsParam := c.Param("workspace")

	ws, err := cs.Workspace.Get("default", wsParam, metav1.GetOptions{})
	if err != nil {
		notFound(c)
		return
	}

	var newState *tfe.StateVersionCreateOptions
	err = jsonapi.UnmarshalPayload(c.Request.Body, newState)
	if err != nil {
		badRequest(c, fmt.Errorf("error unmarshalling state: %s", err))
		return
	}

	secret, err := cs.Secret.Get("default", getStateName(ws.Spec.State), metav1.GetOptions{})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			badRequest(c, fmt.Errorf("error pulling secret %s: %s", getStateName(ws.Spec.State), err))
			return
		}
		secret.Data = make(map[string][]byte)
	}

	secretData, err := b64.StdEncoding.DecodeString(*newState.State)
	if err != nil {
		badRequest(c, fmt.Errorf("error b64 decoding state %s: %s", getStateName(ws.Spec.State), err))
		return
	}

	gzippedData, err := gzipData(secretData)
	if err != nil {
		badRequest(c, fmt.Errorf("error un-gzipping state: %s", err))
		return
	}

	secret.Data["tfstate"] = gzippedData
	_, err = cs.Secret.Update(secret)
	if err != nil {
		badRequest(c, fmt.Errorf("error saving state: %s", err))
		return
	}

	err = jsonapi.MarshalPayload(c.Writer, &tfe.StateVersion{
		Serial:      *newState.Serial,
		DownloadURL: fmt.Sprintf("download/%s/state", wsParam),
	})
	if err != nil {
		logrus.Errorf("error marshalling state version: %s", err)
	}
}
