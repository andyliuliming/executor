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
	// extract to a folder first, then copy to the target first.
	tempFolder, err := ioutil.TempDir("/tmp", "folder_extracted")
	extra := extractor.NewTar()
	err = extra.ExtractStream(tempFolder, reader)
	if err == nil {
		err = v.CopyFolderToAzureShare(tempFolder, storageID, storageSecret, shareName)
		if err == nil {
			v.logger.Info("#########(andliu) ExtractToAzureShare succeeded.", lager.Data{
				"tempFolder": tempFolder,
				"shareName":  shareName})
			return nil
		} else {
			v.logger.Info("#########(andliu) ExtractToAzureShare copy to azure share failed.", lager.Data{"err": err.Error()})
			return err
		}
	} else {
		v.logger.Info("#########(andliu) ExtractToAzureShare extract to temp folder failed.", lager.Data{"err": err.Error()})
		return err
	}
}

func (v *VSync) CopyFolderToAzureShare(src, storageID, storageSecret, shareName string) error {
	v.logger.Info("##########(andliu) copy folder to azure share.", lager.Data{"src": src, "shareName": shareName})
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
		// TODO because 445 port is blocked in microsoft, so we use the proxy to do it...
		options = append(options, "port=444")
		azureFilePath := fmt.Sprintf("//40.112.190.242/%s", shareName) //fmt.Sprintf("//%s.file.core.windows.net/%s", storageID, shareName)
		err = mounter.Mount(azureFilePath, tempFolder, "cifs", options)
		return tempFolder, err
	}
	return "Failed to create temp folder.", err
}
