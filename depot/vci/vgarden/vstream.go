package vgarden

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"code.cloudfoundry.org/executor/depot/vci/helpers/fsync"
	"code.cloudfoundry.org/executor/depot/vci/helpers/mount"
	"code.cloudfoundry.org/executor/depot/vci/vstore"
	"code.cloudfoundry.org/executor/model"
	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/lager"
	"github.com/virtualcloudfoundry/goaci/aci"
)

type VStream struct {
	logger lager.Logger
}

func NewVStream(logger lager.Logger) *VStream {
	return &VStream{
		logger: logger,
	}
}

func (c *VStream) buildVolumes(handle string, bindMounts []garden.BindMount) ([]aci.Volume, []aci.VolumeMount) {
	var volumeMounts []aci.VolumeMount
	var volumes []aci.Volume
	buildInFolders := GetBuldInFolders()
	for _, buildInFolder := range buildInFolders {
		shareName := vstore.BuildShareName(handle, buildInFolder)

		azureFile := &aci.AzureFileVolume{
			ReadOnly:           false,
			ShareName:          shareName,
			StorageAccountName: model.GetExecutorEnvInstance().Config.ContainerProviderConfig.StorageId,
			StorageAccountKey:  model.GetExecutorEnvInstance().Config.ContainerProviderConfig.StorageSecret,
		}
		volume := aci.Volume{
			Name:      shareName,
			AzureFile: azureFile,
		}
		volumes = append(volumes, volume)

		volumeMount := aci.VolumeMount{
			Name:      shareName,
			MountPath: buildInFolder,
			ReadOnly:  false,
		}
		volumeMounts = append(volumeMounts, volumeMount)
	}

	for _, bindMount := range bindMounts {
		alreadyExist := false
		for _, buildInFolder := range buildInFolders {
			if strings.HasPrefix(bindMount.DstPath, buildInFolder) {
				alreadyExist = true
			}
		}
		if !alreadyExist {
			shareName := vstore.BuildShareName(handle, bindMount.DstPath)
			c.logger.Info("#########(andliu) create folder.", lager.Data{
				"handle": handle, "bindMount": bindMount, "shareName": shareName})

			// 1. mount the share created in the virtual diego cell
			// 2. copy all the files in the bindMount.SrcPath to that share.
			// 3. unmount it.
			// merge the folder to the parent.
			azureFile := &aci.AzureFileVolume{
				ReadOnly:           false,
				ShareName:          shareName,
				StorageAccountName: model.GetExecutorEnvInstance().Config.ContainerProviderConfig.StorageId,
				StorageAccountKey:  model.GetExecutorEnvInstance().Config.ContainerProviderConfig.StorageSecret,
			}
			volume := aci.Volume{
				Name:      shareName,
				AzureFile: azureFile,
			}
			volumes = append(volumes, volume)

			volumeMount := aci.VolumeMount{
				Name:      shareName,
				MountPath: bindMount.DstPath,
				ReadOnly:  false,
			}
			volumeMounts = append(volumeMounts, volumeMount)
		}
	}
	return volumes, volumeMounts
}

// 1. provide one api for prepare the azure mounts.
// 2. the /tmp is special before we work out a solution for the common stream in/out method.
// 3. append the /tmp
// 	  a. for each item in the bind mount
//    b. check
func (c *VStream) PrepareVolumeMounts(handle string, bindMounts []garden.BindMount) ([]aci.Volume, []aci.VolumeMount, error) {
	volumes, volumeMounts := c.buildVolumes(handle, bindMounts)
	vs := vstore.NewVStore()
	// 1. prepare the volumes.
	for _, volume := range volumes {
		// create share folder
		err := vs.CreateShareFolder(volume.AzureFile.ShareName)
		if err != nil {
			c.logger.Info("###########(andliu) create share folder failed.", lager.Data{"err": err.Error()})
		}
	}
	// 2. copy the contents.

	mounter := mount.NewMounter()
	for _, bindMount := range bindMounts {
		parentExists := false
		var vol *aci.Volume
		var vm *aci.VolumeMount
		for _, volumeMount := range volumeMounts {
			if strings.HasPrefix(bindMount.DstPath, volumeMount.MountPath) {
				parentExists = true
				vm = &volumeMount

				for _, volume := range volumes {
					if volume.Name == vm.Name {
						vol = &volume
						break
					}
				}

				break
			}
		}

		if !parentExists {
			for _, volumeMount := range volumeMounts {
				if volumeMount.MountPath == bindMount.DstPath {
					vm = &volumeMount
					for _, volume := range volumes {
						if volume.Name == vm.Name {
							vol = &volume
							break
						}
					}
				}
			}
		}

		tempFolder, err := ioutil.TempDir("/tmp", "folder_to_azure_")
		if err == nil {
			options := []string{
				"vers=3.0",
				fmt.Sprintf("username=%s", model.GetExecutorEnvInstance().Config.ContainerProviderConfig.StorageId),
				fmt.Sprintf("password=%s", model.GetExecutorEnvInstance().Config.ContainerProviderConfig.StorageSecret),
				"dir_mode=0777,file_mode=0777,serverino",
			}
			// TODO because 445 port is blocked in microsoft, so we use the proxy to do it...
			options = append(options, "port=444")
			azureFilePath := fmt.Sprintf("//40.112.190.242/%s", vol.AzureFile.ShareName) //fmt.Sprintf("//%s.file.core.windows.net/%s", storageID, shareName)
			err = mounter.Mount(azureFilePath, tempFolder, "cifs", options)
			if err == nil {
				var targetFolder string
				if parentExists {
					// make sub folder.
					// bindMount.DstPath  /tmp/lifecycle
					// vm.MountPath	 /tmp
					rel, err := filepath.Rel(vm.MountPath, bindMount.DstPath)
					if err != nil {
						c.logger.Info("##########(andliu) filepath.Rel failed.", lager.Data{"err": err.Error()})
					}
					targetFolder = filepath.Join(tempFolder, rel)
					err = os.MkdirAll(targetFolder, os.ModeDir)
					if err != nil {
						c.logger.Info("##########(andliu) MkdirAll failed.", lager.Data{"err": err.Error()})
					}
				} else {
					targetFolder = tempFolder
				}
				fsync := fsync.NewFSync()
				err = fsync.CopyFolder(bindMount.SrcPath, targetFolder)
				if err != nil {
					c.logger.Info("##########(andliu) copy folder failed.", lager.Data{
						"parentExists": parentExists,
						"err":          err.Error(),
						"bindMount":    bindMount,
						"tempFolder":   tempFolder})
				}
				mounter.Unmount(tempFolder)
			} else {
				c.logger.Info("##########(andliu) mount temp folder failed.", lager.Data{
					"azureFilePath": azureFilePath,
					"tempFolder":    tempFolder,
				})
			}
		} else {
			c.logger.Info("##########(andliu) create temp folder failed.", lager.Data{"err": err.Error()})
		}
	}

	return volumes, volumeMounts, nil
}

func (c *VStream) appendBindMount(handle string) {

}
