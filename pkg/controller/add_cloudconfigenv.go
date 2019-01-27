package controller

import (
	"github.com/chrsoo/cloud-config-operator/pkg/controller/cloudconfigenv"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, cloudconfigenv.Add)
}
