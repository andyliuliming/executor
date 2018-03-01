package vgarden // import "code.cloudfoundry.org/executor/depot/vci/vgarden"
import (
	"strings"

	"code.cloudfoundry.org/executor/depot/vci/helpers"
	"code.cloudfoundry.org/executor/depot/vci/vstore"

	"code.cloudfoundry.org/executor/model"
	"code.cloudfoundry.org/garden"
	GardenClient "code.cloudfoundry.org/garden/client"
	"code.cloudfoundry.org/lager"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/virtualcloudfoundry/goaci"
	"github.com/virtualcloudfoundry/goaci/aci"
)

type Client interface {
	garden.Client
}

type client struct {
	// connection connection.Connection
	inner     GardenClient.Client
	config    model.ContainerProviderConfig
	aciClient *aci.Client
	logger    lager.Logger
}

// func New() Client {
// 	return &client{
// 	// connection: connection,
// 	}
// }

func NewWithAdapter(gc GardenClient.Client, logger lager.Logger, config model.ContainerProviderConfig) Client {
	var azAuth *goaci.Authentication

	azAuth = goaci.NewAuthentication(azure.PublicCloud.Name, config.ContainerId, config.ContainerSecret, config.SubscriptionId, config.OptionalParam1)

	aciClient, err := aci.NewClient(azAuth)
	if aciClient != nil && err != nil {

	}
	return &client{
		inner:     gc,
		config:    config,
		aciClient: aciClient,
		logger:    logger,
	}
}

func (c *client) Ping() error {
	// c.logger.Info("##########(andliu) ping!!!")
	// containerGroupUsage, err, code := c.aciClient.ListContainerGroupUsage(c.config.Location)
	// if err == nil {
	// 	c.logger.Info("Ping", lager.Data{"code": code, "usage": containerGroupUsage})
	// } else {
	// 	c.logger.Error("Ping", err, lager.Data{"code": code})
	// }
	c.logger.Info("##########(andliu) Ping.")
	return c.inner.Ping()
}

func (c *client) Capacity() (garden.Capacity, error) {
	c.logger.Info("########(andliu) Capacity")
	return c.inner.Capacity()
}

func (c *client) prepareVirtualShares(handle string, bindMounts []garden.BindMount) ([]aci.Volume, []aci.VolumeMount, error) {
	vstore := vstore.NewVStore()
	var volumeMounts []aci.VolumeMount
	var volumes []aci.Volume
	for _, bindMount := range bindMounts {
		shareName, err := vstore.CreateFolder(handle, bindMount.DstPath)
		c.logger.Info("#########(andliu) create folder.", lager.Data{"handle": handle, "bindMount": bindMount})
		if err == nil {
			c.logger.Info("######(andliu) TODO copy to azure share.", lager.Data{"bindMount": bindMount, "ContainerProviderConfig": model.GetExecutorEnvInstance().Config.ContainerProviderConfig})
			// 1. mount the share created in the virtual diego cell
			// 2. copy all the files in the bindMount.SrcPath to that share.
			// 3. unmount it.

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
			vsync := helpers.NewVSync(c.logger)
			err = vsync.CopyFolderToAzureShare(bindMount.SrcPath, azureFile.StorageAccountName, azureFile.StorageAccountKey, azureFile.ShareName)

			volumeMounts = append(volumeMounts, volumeMount)
		} else {
			c.logger.Info("########(andliu) create folder failed.", lager.Data{
				// "handle":      handle,
				// "destination": bindMount.DstPath,
				"err": err.Error()})
			// TODO handle the error case.
		}
	}
	return volumes, volumeMounts, nil
}

func (c *client) Create(spec garden.ContainerSpec) (garden.Container, error) {
	if !strings.HasPrefix(spec.Handle, "executor-healthcheck") {
		c.logger.Info("########(andliu) not health check, so create in aci.", lager.Data{"spec": spec})
		// try to call the create
		var containerGroup aci.ContainerGroup
		executorEnv := model.GetExecutorEnvInstance()
		containerGroup.Location = executorEnv.Config.ContainerProviderConfig.Location
		containerGroup.ContainerGroupProperties.OsType = aci.Linux

		// TODO add the ports.
		var containerProperties aci.ContainerProperties
		containerProperties.Image = "cloudfoundry/cflinuxfs2"

		containerGroup.ContainerGroupProperties.RestartPolicy = aci.OnFailure
		containerProperties.Command = append(containerProperties.Command, "/bin/bash")
		containerProperties.Command = append(containerProperties.Command, "-c")

		const prepareScript = `
	set -e
	echo "#####now /"
	ls /
	echo "#####now /home"
	ls /home
	echo "#####now /home/vcap"
	ls /home/vcap
	echo "#####now remove /home/vcap/app"
	rm -rf /home/vcap/app
	echo "#####check /home/vcap/app gone"
	ls /home/vcap
`
		containerProperties.Command = append(containerProperties.Command, prepareScript)
		// "/bin/bash", "-c"
		if len(spec.NetIn) > 0 {
			containerGroup.IPAddress = &aci.IPAddress{
				Type:  aci.Public,
				Ports: make([]aci.Port, len(spec.NetIn)),
			}
		}
		for _, p := range spec.NetIn {
			containerGroup.IPAddress.Ports =
				append(containerGroup.IPAddress.Ports, aci.Port{
					Protocol: aci.TCP,
					Port:     int32(p.ContainerPort), // TODO use the ContainerPort for all now...
				})
			containerPort := aci.ContainerPort{
				Port:     int32(p.ContainerPort),
				Protocol: aci.ContainerNetworkProtocolTCP,
			}
			containerProperties.Ports = append(containerProperties.Ports, containerPort)
		}
		// containerGroup.ContainerGroupProperties = containerProperties
		// containerProperties.Ports = spec.NetIn
		container := aci.Container{
			Name: spec.Handle,
			// ContainerProperties: aci.ContainerProperties{
			// 	Image:   "cloudfoundry/cflinuxfs2",
			// 	Command: append(container.Command, container.Args...),
			// 	Ports:   make([]aci.ContainerPort, 0, len(container.Ports)),
			// },
		}
		containerProperties.Resources.Requests.CPU = 1          // hard code 1
		containerProperties.Resources.Requests.MemoryInGB = 0.3 // hard code memory 1

		containerProperties.Resources.Limits.CPU = 1          // hard code 1
		containerProperties.Resources.Limits.MemoryInGB = 0.3 // hard code memory 1

		// prepare the share folder to be mounted
		handle := spec.Handle
		// handle = "vgarden" // TODO remove this, hard code for consistent folder.
		volumes, volumeMounts, err := c.prepareVirtualShares(handle, spec.BindMounts)
		c.logger.Info("###########(andliu) prepareVirtualShares result.",
			lager.Data{"volumes": volumes, "volumeMounts": volumeMounts})
		if err == nil {
			containerGroup.ContainerGroupProperties.Volumes = volumes
			containerProperties.VolumeMounts = volumeMounts
		} else {
			// handle this error case
			c.logger.Info("##########(andliu) prepare virtual shares failed.", lager.Data{"err": err.Error()})
		}
		// spec.

		container.ContainerProperties = containerProperties
		containerGroup.Containers = append(containerGroup.Containers, container)

		// hard code a resource group name here.
		containerGroupCreated, err := c.aciClient.CreateContainerGroup(executorEnv.ResourceGroup, spec.Handle, containerGroup)
		if err == nil {
			// TODO wait for the command exit.
			c.logger.Info("###########(andliu) createcontainergroup succeeded.", lager.Data{"containerGroupCreated": containerGroupCreated})
		} else {
			c.logger.Info("###########(andliu) CreateContainerGroup failed.", lager.Data{"err": err.Error()})
		}
	}

	return c.inner.Create(spec)
}

func (c *client) Containers(properties garden.Properties) ([]garden.Container, error) {
	// handles, err := client.connection.List(properties)
	// if err != nil {
	// 	return nil, err
	// }

	// containers := []garden.Container{}
	// for _, handle := range handles {
	// 	containers = append(containers, newContainer(handle, client.connection))
	// }

	// we only support get all containers.
	c.logger.Info("########(andliu) Containers", lager.Data{"properties": properties})
	return c.inner.Containers(properties)
}

func (c *client) Destroy(handle string) error {
	// err := client.connection.Destroy(handle)

	c.logger.Info("########(andliu) Destroy", lager.Data{"handle": handle})
	return c.inner.Destroy(handle)
}

func (c *client) BulkInfo(handles []string) (map[string]garden.ContainerInfoEntry, error) {

	// return map[string]garden.ContainerInfoEntry{}, nil
	// return client.connection.BulkInfo(handles)

	c.logger.Info("########(andliu) BulkInfo", lager.Data{"handles": handles})
	return c.inner.BulkInfo(handles)
}

func (c *client) BulkMetrics(handles []string) (map[string]garden.ContainerMetricsEntry, error) {
	// return map[string]garden.ContainerMetricsEntry{}, nil

	c.logger.Info("########(andliu) BulkMetrics", lager.Data{"handles": handles})
	return c.inner.BulkMetrics(handles)
}

func (c *client) Lookup(handle string) (garden.Container, error) {
	// handles, err := client.connection.List(nil)
	// if err != nil {
	// 	return nil, err
	// }

	// for _, h := range handles {
	// 	if h == handle {
	// 		return newContainer(handle, client.connection), nil
	// 	}
	// }

	// return nil, garden.ContainerNotFoundError{Handle: handle}
	// return nil, nil
	c.logger.Info("########(andliu) Lookup", lager.Data{"handle": handle})
	return c.inner.Lookup(handle)
}
