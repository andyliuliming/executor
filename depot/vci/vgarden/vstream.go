package vgarden

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"code.cloudfoundry.org/archiver/extractor"
	"code.cloudfoundry.org/executor/depot/vci/helpers/fsync"
	"code.cloudfoundry.org/executor/depot/vci/helpers/mount"
	"code.cloudfoundry.org/executor/depot/vci/vstore"
	"code.cloudfoundry.org/executor/model"
	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/lager"
	"github.com/Azure/go-autorest/autorest/azure"
	uuid "github.com/satori/go.uuid"
	"github.com/virtualcloudfoundry/goaci"
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

func (c *VStream) GetContainerSwapRootShareFolder(handle string) string {
	shareName := fmt.Sprintf("root-%s", handle)
	return shareName
}

func (c *VStream) MountContainerRoot(handle string) (string, error) {
	shareName := c.GetContainerSwapRootShareFolder(handle)
	vs := vstore.NewVStore()
	// 1. prepare the volumes.
	// create share folder
	err := vs.CreateShareFolder(shareName)
	c.logger.Info("#########(andliu) create share folder, will fail second time.")
	mountedRootFolder, err := ioutil.TempDir("/tmp", "folder_to_azure_")
	options := []string{
		"vers=3.0",
		fmt.Sprintf("username=%s", model.GetExecutorEnvInstance().Config.ContainerProviderConfig.StorageId),
		fmt.Sprintf("password=%s", model.GetExecutorEnvInstance().Config.ContainerProviderConfig.StorageSecret),
		"dir_mode=0777,file_mode=0777,serverino",
	}
	// TODO because 445 port is blocked in microsoft, so we use the proxy to do it...
	options = append(options, "port=444")
	azureFilePath := fmt.Sprintf("//40.112.190.242/%s", shareName) //fmt.Sprintf("//%s.file.core.windows.net/%s", storageID, shareName)
	mounter := mount.NewMounter()
	err = mounter.Mount(azureFilePath, mountedRootFolder, "cifs", options)
	if err != nil {
		c.logger.Info("#######(andliu) PrepareSwapVolumeMount mount failed.", lager.Data{
			"src":  azureFilePath,
			"dest": mountedRootFolder,
			"err":  err.Error()})
		return "", err
	}
	return mountedRootFolder, nil
}

func (c *VStream) PrepareSwapVolumeMount(handle string, bindMounts []garden.BindMount) ([]aci.Volume, []aci.VolumeMount, error) {
	var volumeMounts []aci.VolumeMount
	var volumes []aci.Volume
	// shareName := vstore.BuildShareName(handle, buildInFolder)
	shareName := c.GetContainerSwapRootShareFolder(handle)

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
		MountPath: GetSwapRoot(),
		ReadOnly:  false,
	}
	volumeMounts = append(volumeMounts, volumeMount)

	mountedRootFolder, err := c.MountContainerRoot(handle)
	options := []string{
		"vers=3.0",
		fmt.Sprintf("username=%s", model.GetExecutorEnvInstance().Config.ContainerProviderConfig.StorageId),
		fmt.Sprintf("password=%s", model.GetExecutorEnvInstance().Config.ContainerProviderConfig.StorageSecret),
		"dir_mode=0777,file_mode=0777,serverino",
	}
	// TODO because 445 port is blocked in microsoft, so we use the proxy to do it...
	options = append(options, "port=444")
	azureFilePath := fmt.Sprintf("//40.112.190.242/%s", volume.AzureFile.ShareName) //fmt.Sprintf("//%s.file.core.windows.net/%s", storageID, shareName)
	mounter := mount.NewMounter()
	err = mounter.Mount(azureFilePath, mountedRootFolder, "cifs", options)
	if err != nil {
		c.logger.Info("#######(andliu) PrepareSwapVolumeMount mount failed.", lager.Data{
			"src":  azureFilePath,
			"dest": mountedRootFolder,
			"err":  err.Error()})
		return nil, nil, err
	}

	f, err := os.Create(filepath.Join(mountedRootFolder, "post_task.sh"))
	f.WriteString("#!/bin/bash\n")

	for _, bindMount := range bindMounts {
		fsync := fsync.NewFSync(c.logger)
		relativePath := fmt.Sprintf(".%s", bindMount.DstPath) // ./tmp/app
		targetFolder := filepath.Join(mountedRootFolder, relativePath)
		err = fsync.CopyFolder(bindMount.SrcPath, targetFolder)
		if err != nil {
			c.logger.Info("#######(andliu) PrepareSwapVolumeMount copy folder failed.", lager.Data{
				"src":  bindMount.SrcPath,
				"dest": targetFolder,
				"err":  err.Error(),
			})
		}
		postCopyTask := fmt.Sprintf("rsync -a %s/ %s\n", filepath.Join(GetSwapRoot(), relativePath), bindMount.DstPath)
		c.logger.Info("########(andliu) postCopyTaskLine.", lager.Data{"line": postCopyTask})
		f.WriteString(postCopyTask)
		// c.logger.Info("########(andliu) postCopyTaskLine.", lager.Data{"line": postCopyTask})

	}
	f.Close()
	err = mounter.Unmount(mountedRootFolder)
	if err != nil {
		c.logger.Info("#######(andliu) umount failed.", lager.Data{"err": err.Error(), "folder": mountedRootFolder})
	}
	return volumes, volumeMounts, nil
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

		//?
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
				"dir_mode=0777,file_mode=0777,mfsymlinks,serverino",
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
				fsync := fsync.NewFSync(c.logger)
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

// assume that the container already there.
func (c *VStream) StreamIn(handle, destination string, reader io.ReadCloser) error {
	// extract the tar to the target model.To
	// TODO create one share folder for /tmp
	// 1. get the container configs.
	c.logger.Info("#########(andliu) VStream StreamIn starts.", lager.Data{"handle": handle, "dest": destination})
	var finaldestination string
	if destination == "." {
		// TODO: workaround, we guess . is the /home/vcap.
		// will extract the droplet file to this folder.
		finaldestination = "/home/vcap"
	} else {
		finaldestination = destination
	}
	// handle := step.container.Handle()
	extractedFolder, err := ioutil.TempDir("/tmp", "folder_extracted_")
	extra := extractor.NewTar()
	err = extra.ExtractStream(extractedFolder, reader)

	mounter := mount.NewMounter()
	mountedRootFolder, err := c.MountContainerRoot(handle)
	fsync := fsync.NewFSync(c.logger)
	id, _ := uuid.NewV4()
	subfolder := id.String()
	c.logger.Info("##########(andliu) subfolder name is.", lager.Data{"subfolder": subfolder})
	err = fsync.CopyFolder(extractedFolder, filepath.Join(mountedRootFolder, subfolder))
	if err != nil {
		c.logger.Info("########(andliu) copy failed.", lager.Data{
			"err":  err.Error(),
			"src":  extractedFolder,
			"dest": mountedRootFolder})
	}
	f, err := os.Open(filepath.Join(mountedRootFolder, "post_task.sh"))
	f.WriteString("#!/bin/bash\n")

	postCopyTask := fmt.Sprintf("rsync -a %s/ %s\n", filepath.Join(GetSwapRoot(), subfolder), finaldestination)
	c.logger.Info("########(andliu) postCopyTask.", lager.Data{"postCopyTask": postCopyTask})
	f.WriteString(postCopyTask)
	f.Close()
	mounter.Unmount(mountedRootFolder)
	var azAuth *goaci.Authentication
	executorEnv := model.GetExecutorEnvInstance()
	config := executorEnv.Config.ContainerProviderConfig
	azAuth = goaci.NewAuthentication(azure.PublicCloud.Name, config.ContainerId, config.ContainerSecret, config.SubscriptionId, config.OptionalParam1)

	aciClient, err := aci.NewClient(azAuth)
	containerGroupGot, err, _ := aciClient.GetContainerGroup(executorEnv.ResourceGroup, handle)
	if err == nil {
		for idx, _ := range containerGroupGot.ContainerGroupProperties.Volumes {
			containerGroupGot.ContainerGroupProperties.Volumes[idx].AzureFile.StorageAccountKey =
				executorEnv.Config.ContainerProviderConfig.StorageSecret
		}
		c.logger.Info("#########(andliu) update container group:", lager.Data{"containerGroupGot": *containerGroupGot})
		containerGroupUpdated, err := aciClient.UpdateContainerGroup(executorEnv.ResourceGroup, handle, *containerGroupGot)
		retry := 0
		for err != nil && retry < 10 {
			c.logger.Info("#########(andliu) update container group failed.", lager.Data{"err": err.Error()})
			time.Sleep(60 * time.Second)
			containerGroupUpdated, err = aciClient.UpdateContainerGroup(executorEnv.ResourceGroup, handle, *containerGroupGot)
			retry++
		}
		if err == nil {
			c.logger.Info("##########(andliu) update container group succeeded.", lager.Data{"containerGroupUpdated": containerGroupUpdated})
		} else {
			c.logger.Info("#########(andliu) update container group failed.", lager.Data{"err": err.Error()})
		}
	} else {
		c.logger.Info("#########(andliu) StreamIn GetContainerGroup failed.", lager.Data{"handle": handle, "dest": destination})
	}
	return err
	// vol, vm, parentExist, err := c.appendBindMount(handle, finaldestination)
	// if err != nil {
	// 	return err
	// }
	// vsync := helpers.NewVSync(c.logger)
	// err = vsync.ExtractToAzureShare(reader, vol, vm, parentExist, destination)

	// if !parentExist {
	// 	var azAuth *goaci.Authentication

	// 	executorEnv := model.GetExecutorEnvInstance()
	// 	config := executorEnv.Config.ContainerProviderConfig
	// 	azAuth = goaci.NewAuthentication(azure.PublicCloud.Name, config.ContainerId, config.ContainerSecret, config.SubscriptionId, config.OptionalParam1)

	// 	aciClient, err := aci.NewClient(azAuth)
	// 	if err == nil {
	// 		containerGroupGot, err, _ := aciClient.GetContainerGroup(executorEnv.ResourceGroup, handle)
	// 		if err == nil {
	// 			containerGroupGot.ContainerGroupProperties.Volumes = append(
	// 				containerGroupGot.ContainerGroupProperties.Volumes, *vol)

	// 			for idx, _ := range containerGroupGot.ContainerGroupProperties.Volumes {
	// 				containerGroupGot.ContainerGroupProperties.Volumes[idx].AzureFile.StorageAccountKey =
	// 					executorEnv.Config.ContainerProviderConfig.StorageSecret
	// 			}

	// 			for idx, _ := range containerGroupGot.ContainerGroupProperties.Containers {
	// 				containerGroupGot.ContainerGroupProperties.Containers[idx].VolumeMounts = append(
	// 					containerGroupGot.ContainerGroupProperties.Containers[idx].VolumeMounts, *vm)
	// 			}

	// 			// newVolume := aci.Volume{Name: shareName, AzureFile: azureFile}
	// 			c.logger.Info("#########(andliu) update container group:", lager.Data{"containerGroupGot": *containerGroupGot})
	// 			containerGroupUpdated, err := aciClient.UpdateContainerGroup(executorEnv.ResourceGroup, handle, *containerGroupGot)
	// 			retry := 0
	// 			for err != nil && retry < 10 {
	// 				c.logger.Info("#########(andliu) update container group failed.", lager.Data{"err": err.Error()})
	// 				time.Sleep(60 * time.Second)
	// 				containerGroupUpdated, err = aciClient.UpdateContainerGroup(executorEnv.ResourceGroup, handle, *containerGroupGot)
	// 				retry++
	// 			}
	// 			if err == nil {
	// 				c.logger.Info("##########(andliu) update container group succeeded.", lager.Data{"containerGroupUpdated": containerGroupUpdated})
	// 			} else {
	// 				c.logger.Info("#########(andliu) update container group failed.", lager.Data{"err": err.Error()})
	// 			}
	// 		} else {
	// 			c.logger.Info("#########(andliu) get container group failed.", lager.Data{"err": err.Error()})
	// 		}
	// 	}
	// }

	// vsync := helpers.NewVSync(c.logger)

	// handle := step.container.Handle()
	// c.logger.Info("##########(andliu) perform vStreamIn step.", lager.Data{
	// 	"handle":      handle,
	// 	"destination": finaldestination})
	// var azAuth *goaci.Authentication

	// executorEnv := model.GetExecutorEnvInstance()
	// config := executorEnv.Config.ContainerProviderConfig
	// azAuth = goaci.NewAuthentication(azure.PublicCloud.Name, config.ContainerId, config.ContainerSecret, config.SubscriptionId, config.OptionalParam1)

	// aciClient, err := aci.NewClient(azAuth)
	// if err == nil {
	// 	containerGroupGot, err, _ := aciClient.GetContainerGroup(executorEnv.ResourceGroup, handle)
	// 	if err == nil {
	// 		c.logger.Info("##########(andliu) download step in get container group.",
	// 			lager.Data{
	// 				"handle":            handle,
	// 				"destination":       finaldestination,
	// 				"containerGroupGot": *containerGroupGot})

	// 		// create a folder
	// 		vstore := vstore.NewVStore()
	// 		shareName, err := vstore.CreateFolder(handle, finaldestination)

	// 		executorEnv := model.GetExecutorEnvInstance()
	// 		if err == nil {
	// 			c.logger.Info("#########(andliu) shareName.", lager.Data{"shareName": shareName})
	// 			azureFile := &aci.AzureFileVolume{
	// 				ReadOnly:           false,
	// 				ShareName:          shareName,
	// 				StorageAccountName: executorEnv.Config.ContainerProviderConfig.StorageId,
	// 				StorageAccountKey:  executorEnv.Config.ContainerProviderConfig.StorageSecret,
	// 			}
	// 			newVolume := aci.Volume{Name: shareName, AzureFile: azureFile}
	// 			containerGroupGot.ContainerGroupProperties.Volumes = append(
	// 				containerGroupGot.ContainerGroupProperties.Volumes, newVolume)
	// 			volumeMount := aci.VolumeMount{
	// 				Name:      shareName,
	// 				MountPath: finaldestination,
	// 				ReadOnly:  false,
	// 			}
	// 			vsync := helpers.NewVSync(c.logger)
	// 			// TODO check whether there's already parent folder mounted.
	// 			// if yes, then no need to mount ,just mount the parent, and copy.
	// 			// if no, create a new folder to map.
	// 			err = vsync.ExtractToAzureShare(reader, azureFile.StorageAccountName, azureFile.StorageAccountKey, azureFile.ShareName)
	// 			if err == nil {
	// 				// save back the storage account key
	// 				for idx, _ := range containerGroupGot.ContainerGroupProperties.Volumes {
	// 					containerGroupGot.ContainerGroupProperties.Volumes[idx].AzureFile.StorageAccountKey =
	// 						executorEnv.Config.ContainerProviderConfig.StorageSecret
	// 				}
	// 				for idx, _ := range containerGroupGot.ContainerGroupProperties.Containers {
	// 					containerGroupGot.ContainerGroupProperties.Containers[idx].VolumeMounts = append(
	// 						containerGroupGot.ContainerGroupProperties.Containers[idx].VolumeMounts, volumeMount)
	// 				}
	// 				c.logger.Info("#########(andliu) update container group:", lager.Data{"containerGroupGot": *containerGroupGot})
	// 				containerGroupUpdated, err := aciClient.UpdateContainerGroup(executorEnv.ResourceGroup, handle, *containerGroupGot)
	// 				retry := 0
	// 				for err != nil && retry < 10 {
	// 					c.logger.Info("#########(andliu) update container group failed.", lager.Data{"err": err.Error()})
	// 					time.Sleep(60 * time.Second)
	// 					containerGroupUpdated, err = aciClient.UpdateContainerGroup(executorEnv.ResourceGroup, handle, *containerGroupGot)
	// 					retry++
	// 				}
	// 				if err == nil {
	// 					c.logger.Info("##########(andliu) update container group succeeded.", lager.Data{"containerGroupUpdated": containerGroupUpdated})
	// 				} else {
	// 					c.logger.Info("#########(andliu) update container group failed.", lager.Data{"err": err.Error()})
	// 				}
	// 			} else {
	// 				c.logger.Info("########(andliu) extract to azure share failed.", lager.Data{"err": err.Error()})
	// 			}
	// 		} else {
	// 			c.logger.Info("#########(andliu) shareName failed.", lager.Data{"err": err.Error()})
	// 		}
	// 	} else {
	// 		c.logger.Info("##########(andliu) GetContainerGroup.", lager.Data{"err": err.Error()})
	// 	}
	// } else {
	// 	c.logger.Info("##########(andliu) new client.", lager.Data{"err": err.Error()})
	// }
	// return err
}

func (c *VStream) appendBindMount(handle, destination string) (*aci.Volume, *aci.VolumeMount, bool, error) {
	var azAuth *goaci.Authentication

	executorEnv := model.GetExecutorEnvInstance()
	config := executorEnv.Config.ContainerProviderConfig
	azAuth = goaci.NewAuthentication(azure.PublicCloud.Name, config.ContainerId, config.ContainerSecret, config.SubscriptionId, config.OptionalParam1)

	aciClient, err := aci.NewClient(azAuth)
	var vol *aci.Volume
	var vm *aci.VolumeMount
	parentExists := false
	if err == nil {
		containerGroup, err, _ := aciClient.GetContainerGroup(executorEnv.ResourceGroup, handle)
		if err != nil {
			c.logger.Info("########(andliu) failed to get container.", lager.Data{"handle": handle, "err": err.Error()})
			return nil, nil, false, err
		}
		volumes := containerGroup.Volumes
		// make assumption that each container group have only one.
		volumeMounts := containerGroup.Containers[0].VolumeMounts

		for _, volumeMount := range volumeMounts {
			if strings.HasPrefix(destination, volumeMount.MountPath) {
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
			// we need to create new mount
			shareName := vstore.BuildShareName(handle, destination)

			azureFile := &aci.AzureFileVolume{
				ReadOnly:           false,
				ShareName:          shareName,
				StorageAccountName: model.GetExecutorEnvInstance().Config.ContainerProviderConfig.StorageId,
				StorageAccountKey:  model.GetExecutorEnvInstance().Config.ContainerProviderConfig.StorageSecret,
			}
			vol = &aci.Volume{
				Name:      shareName,
				AzureFile: azureFile,
			}
			// volumes = append(volumes, vol)

			vm = &aci.VolumeMount{
				Name:      shareName,
				MountPath: destination,
				ReadOnly:  false,
			}
			// volumeMounts = append(volumeMounts, volumeMount)
		}
	} else {
		c.logger.Info("#########(andliu) new client failed.", lager.Data{"err": err.Error()})
	}
	return vol, vm, parentExists, nil
}
