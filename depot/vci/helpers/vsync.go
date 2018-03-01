package helpers // import "code.cloudfoundry.org/executor/depot/vci/helpers"

import (
	"fmt"
	"io"
	"io/ioutil"

	"code.cloudfoundry.org/archiver/extractor"

	"code.cloudfoundry.org/executor/depot/vci/helpers/fsync"
	"code.cloudfoundry.org/executor/depot/vci/helpers/mount"
	"code.cloudfoundry.org/lager"
)

// copy one folder to one azure share.

type VSync struct {
	logger lager.Logger
}

func NewVSync(logger lager.Logger) *VSync {
	return &VSync{
		logger: logger,
	}
}

func (v *VSync) ExtractToAzureShare(reader io.ReadCloser, storageID, storageSecret, shareName string) error {
	v.logger.Info("#########(andliu) ExtractToAzureShare try to extract to azure share.")
	mounter := mount.NewMounter()
	tempFolder, err := v.mountToTempFolder(storageID, storageSecret, shareName)

	extra := extractor.NewTar()
	if err == nil {
		err = extra.ExtractStream(tempFolder, reader)
		if err == nil {
			mounter.Unmount(tempFolder)
			return nil
		} else {
			v.logger.Info("#########(andliu) ExtractToAzureShare ExtractStream to azure share failed.", lager.Data{"err": err.Error()})
			return err
		}
	} else {
		v.logger.Info("#########(andliu) ExtractToAzureShare extract to azure share failed.", lager.Data{"err": err.Error()})
		return err
	}
}

func (v *VSync) CopyFolderToAzureShare(src, storageID, storageSecret, shareName string) error {
	// vers=<smb-version>,username=<storage-account-name>,password=<storage-account-key>,dir_mode=0777,file_mode=0777
	//,serverino
	mounter := mount.NewMounter()
	tempFolder, err := v.mountToTempFolder(storageID, storageSecret, shareName)
	if err == nil {
		fsync := fsync.NewFSync()
		err = fsync.CopyFolder(src, tempFolder)
		if err == nil {
			mounter.Unmount(tempFolder)
			return nil
		} else {
			return err
		}
	} else {
		return err
	}
}

func (v *VSync) mountToTempFolder(storageID, storageSecret, shareName string) (string, error) {
	options := []string{
		"vers=3.0",
		fmt.Sprintf("username=%s", storageID),
		fmt.Sprintf("password=%s", storageSecret),
		"dir_mode=0777,file_mode=0777,serverino",
	}

	mounter := mount.NewMounter()
	tempFolder, err := ioutil.TempDir("/tmp", "folder_to_azure")
	if err == nil {
		azureFilePath := fmt.Sprintf("//%s.file.core.windows.net/%s", storageID, storageSecret)
		err = mounter.Mount(azureFilePath, tempFolder, "cifs", options)
		return tempFolder, err
	}
	return "Failed to craete temp folder.", err
}
