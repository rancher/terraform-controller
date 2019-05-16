package terraform

import (
	//v1 "github.com/rancher/terraform-controller/pkg/apis/terraformcontroller.cattle.io/v1"
	//terraformFakes "github.com/rancher/terraform-controller/pkg/generated/controllers/terraformcontroller.cattle.io/v1/fakes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFooControllerOnChange(t *testing.T) {
	assert := assert.New(t)
	assert.True(true)
}

func TestFooControllerOnRemove(t *testing.T) {
	assert := assert.New(t)
	assert.True(true)
}
