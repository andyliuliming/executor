package vstore

import (
	"crypto/sha1"
	"fmt"

	"code.cloudfoundry.org/executor/model"
	"github.com/Azure/go-autorest/autorest/azure"
)

// provide a global file share service
// 1. get by container id and relative path
// 2. create by container id and relative path.
type VStore struct {
	AzureFileClient AzureFileClient
}

func NewVStore() *VStore {
	azureFileClient := AzureFileClient{env: azure.PublicCloud}

	return &VStore{
		AzureFileClient: azureFileClient,
	}
}

func (vs *VStore) buildShareName(containerId string, path string) string {
	// return containerId.path
	h := sha1.New()
	originStr := fmt.Sprintf("%s-%s", containerId, path)
	h.Write([]byte(originStr))
	bs := h.Sum(nil)
	// return fmt.Sprintf("%x", bs)
	return fmt.Sprintf("%s-%x", containerId, bs) // TODO use the full hash
}

func (vs *VStore) CreateFolder(containerId string, path string) (string, error) {
	shareName := vs.buildShareName(containerId, path)
	err := vs.createShareFolder(shareName)
	return shareName, err
}

func (vs *VStore) createShareFolder(name string) error {
	providerConfig := model.GetExecutorEnvInstance().Config.ContainerProviderConfig
	return vs.AzureFileClient.createFileShare(providerConfig.StorageId, providerConfig.StorageSecret, name, 50)
}
