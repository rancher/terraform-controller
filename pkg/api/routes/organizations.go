package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/google/jsonapi"
	"github.com/hashicorp/go-tfe"
	"github.com/sirupsen/logrus"
)

func entitlement(c *gin.Context) {
	c.Header("Content-Type", jsonapi.MediaType)
	err := jsonapi.MarshalPayload(c.Writer, &tfe.Entitlements{
		Operations: true,
	})
	if err != nil {
		logrus.Errorf("error marshalling entitle: %s", err)
	}
}
