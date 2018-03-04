package vgarden // import "code.cloudfoundry.org/executor/depot/vci/vgarden"
import (
	"strings"

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
	c.logger.Info("##########(andliu) Ping.")
	return c.inner.Ping()
}

func (c *client) Capacity() (garden.Capacity, error) {
	c.logger.Info("########(andliu) Capacity")
	return c.inner.Capacity()
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
	echo "#####ls /tmp"
	ls /tmp
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
		if len(spec.NetIn) > 0 {
			containerGroup.IPAddress = &aci.IPAddress{
				Type:  aci.Public,
				Ports: make([]aci.Port, 0),
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
		container := aci.Container{
			Name: spec.Handle,
		}
		containerProperties.Resources.Requests.CPU = 1          // hard code 1
		containerProperties.Resources.Requests.MemoryInGB = 0.3 // hard code memory 1

		containerProperties.Resources.Limits.CPU = 1          // hard code 1
		containerProperties.Resources.Limits.MemoryInGB = 0.3 // hard code memory 1

		// prepare the share folder to be mounted
		handle := spec.Handle
		// we need to merge the bindMounts together
		vst := NewVStream(c.logger)
		volumes, volumeMounts, err := vst.PrepareVolumeMounts(handle, spec.BindMounts)
		c.logger.Info("###########(andliu) prepareVirtualShares result.",
			lager.Data{"volumes": volumes, "volumeMounts": volumeMounts})
		if err == nil {
			containerGroup.ContainerGroupProperties.Volumes = volumes
			containerProperties.VolumeMounts = volumeMounts
		} else {
			// handle this error case
			c.logger.Info("##########(andliu) prepare virtual shares failed.", lager.Data{"err": err.Error()})
		}

		container.ContainerProperties = containerProperties
		containerGroup.Containers = append(containerGroup.Containers, container)

		// hard code a resource group name here.
		containerGroupCreated, err := c.aciClient.CreateContainerGroup(executorEnv.ResourceGroup, spec.Handle, containerGroup)
		if err == nil {
			// TODO wait for the command exit.
			c.logger.Info("###########(andliu) createcontainergroup succeeded.", lager.Data{"containerGroupCreated": containerGroupCreated})
		} else {
			c.logger.Info("###########(andliu) CreateContainerGroup failed.", lager.Data{
				"err":            err.Error(),
				"containerGroup": containerGroup})
		}
	}

	return c.inner.Create(spec)
}

func (c *client) Containers(properties garden.Properties) ([]garden.Container, error) {
	// we only support get all containers.
	c.logger.Info("########(andliu) Containers", lager.Data{"properties": properties})
	return c.inner.Containers(properties)
}

func (c *client) Destroy(handle string) error {
	c.logger.Info("########(andliu) Destroy", lager.Data{"handle": handle})
	return c.inner.Destroy(handle)
}

func (c *client) BulkInfo(handles []string) (map[string]garden.ContainerInfoEntry, error) {
	return c.inner.BulkInfo(handles)
}

func (c *client) BulkMetrics(handles []string) (map[string]garden.ContainerMetricsEntry, error) {
	return c.inner.BulkMetrics(handles)
}

func (c *client) Lookup(handle string) (garden.Container, error) {
	c.logger.Info("########(andliu) Lookup", lager.Data{"handle": handle})
	return c.inner.Lookup(handle)
}
