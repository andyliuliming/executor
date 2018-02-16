package aci // import "code.cloudfoundry.org/executor/depot/vci/aci"

import (
	"github.com/Azure/go-autorest/autorest/azure"

	goaci "github.com/virtualcloudfoundry/goaci"
	client "github.com/virtualcloudfoundry/goaci/aci"
)

type AciAdapter struct {
	AciClient *client.Client
}

func NewAciAdapter() (aci *AciAdapter, err error) {
	var azAuth *goaci.Authentication
	azAuth = goaci.NewAuthentication(azure.PublicCloud.Name, "", "", "", "")
	aciClient, err := client.NewClient(azAuth)
	aciAdapter := &AciAdapter{
		AciClient: aciClient,
	}
	return aciAdapter, err
}
