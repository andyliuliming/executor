package vgarden

import (
	"io"
	"strings"
	"time"

	"code.cloudfoundry.org/executor/model"
	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/lager"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/virtualcloudfoundry/goaci"
	"github.com/virtualcloudfoundry/goaci/aci"
)

type VContainer struct {
	inner  garden.Container
	logger lager.Logger
}

func NewVContainer(inner garden.Container, logger lager.Logger) garden.Container {
	return &VContainer{
		inner:  inner,
		logger: logger,
	}
}

func (container *VContainer) Handle() string {
	return container.inner.Handle()
	// return container.handle
}

func (container *VContainer) Stop(kill bool) error {
	return container.inner.Stop(kill)
	// return container.connection.Stop(container.handle, kill)
}

func (container *VContainer) Info() (garden.ContainerInfo, error) {
	return container.inner.Info()
	// return garden.ContainerInfo{}, nil
	// return container.connection.Info(container.handle)
}

func (container *VContainer) StreamIn(spec garden.StreamInSpec) error {
	// TODO move share creation logic to here.
	return container.inner.StreamIn(spec)
}

func (container *VContainer) StreamOut(spec garden.StreamOutSpec) (io.ReadCloser, error) {
	return container.inner.StreamOut(spec)
	// return nil, nil
	// return container.connection.StreamOut(container.handle, spec)
}

func (container *VContainer) CurrentBandwidthLimits() (garden.BandwidthLimits, error) {
	// return garden.BandwidthLimits{}, nil
	return container.inner.CurrentBandwidthLimits()
	// return container.connection.CurrentBandwidthLimits(container.handle)
}

func (container *VContainer) CurrentCPULimits() (garden.CPULimits, error) {
	// return container.connection.CurrentCPULimits(container.handle)
	return container.inner.CurrentCPULimits()
	// return garden.CPULimits{}, nil
}

func (container *VContainer) CurrentDiskLimits() (garden.DiskLimits, error) {
	// return container.connection.CurrentDiskLimits(container.handle)
	return container.inner.CurrentDiskLimits()
	// return garden.DiskLimits{}, nil
}

func (container *VContainer) CurrentMemoryLimits() (garden.MemoryLimits, error) {
	// return container.connection.CurrentMemoryLimits(container.handle)
	return container.inner.CurrentMemoryLimits()
	// return garden.MemoryLimits{}, nil
}

func (container *VContainer) Run(spec garden.ProcessSpec, io garden.ProcessIO) (garden.Process, error) {
	// return container.connection.Run(container.handle, spec, io)
	container.logger.Info("#########(andliu) vcontainer.go L81 run container with spec:", lager.Data{"spec": spec})
	var azAuth *goaci.Authentication

	executorEnv := model.GetExecutorEnvInstance()
	config := executorEnv.Config.ContainerProviderConfig
	azAuth = goaci.NewAuthentication(azure.PublicCloud.Name, config.ContainerId, config.ContainerSecret, config.SubscriptionId, config.OptionalParam1)

	aciClient, err := aci.NewClient(azAuth)
	if err == nil {
		containerGroupGot, err, code := aciClient.GetContainerGroup(executorEnv.ResourceGroup, container.inner.Handle())
		if err != nil {
			for idx, _ := range containerGroupGot.ContainerGroupProperties.Volumes {
				containerGroupGot.ContainerGroupProperties.Volumes[idx].AzureFile.StorageAccountKey =
					executorEnv.Config.ContainerProviderConfig.StorageSecret
			}

			for idx, _ := range containerGroupGot.Containers {
				for _, envStr := range spec.Env {
					splits := strings.Split(envStr, "=")
					containerGroupGot.Containers[idx].ContainerProperties.EnvironmentVariables =
						append(containerGroupGot.Containers[idx].ContainerProperties.EnvironmentVariables,
							aci.EnvironmentVariable{Name: splits[0], Value: splits[1]})

					containerGroupGot.Containers[idx].Command = []string{"env"}
				}
				container.logger.Info("###########(andliu) prepare commands.", lager.Data{"path": spec.Path, "args": spec.Args})
				// containerGroupGot.Containers[idx].Command = append(containerGroupGot.Containers[idx].Command, spec.Path)
				// for _, para := range spec.Args {
				// 	containerGroupGot.Containers[idx].Command = append(containerGroupGot.Containers[idx].Command, para)
				// }
			}
			// prepare the commands.
			container.logger.Info("#########(andliu) container group got.", lager.Data{"containerGroupGot": *containerGroupGot})
			aciClient.UpdateContainerGroup(executorEnv.ResourceGroup, container.inner.Handle(), *containerGroupGot)
		} else {
			container.logger.Info("#########(andliu) vcontainer.go L92 got container in vcontainer.",
				lager.Data{"err": err.Error(), "code": code})
		}
	} else {
		container.logger.Info("########(andliu) Run in VContainer failed.", lager.Data{"err": err.Error()})
	}
	return container.inner.Run(spec, io)
}

func (container *VContainer) Attach(processID string, io garden.ProcessIO) (garden.Process, error) {
	// return container.connection.Attach(container.handle, processID, io)
	return container.inner.Attach(processID, io)
	// return nil, nil
}

func (container *VContainer) NetIn(hostPort, containerPort uint32) (uint32, uint32, error) {
	// return 0, 0, container.connection.NetIn(container.handle, hostPort, containerPort)
	return container.inner.NetIn(hostPort, containerPort)
	// return 0, 0, nil
}

func (container *VContainer) NetOut(netOutRule garden.NetOutRule) error {
	// return container.connection.NetOut(container.handle, netOutRule)
	return container.inner.NetOut(netOutRule)
	// return nil
}

func (container *VContainer) BulkNetOut(netOutRules []garden.NetOutRule) error {
	// return container.connection.BulkNetOut(container.handle, netOutRules)
	return container.inner.BulkNetOut(netOutRules)
	// return nil
}

func (container *VContainer) Metrics() (garden.Metrics, error) {
	// return container.connection.Metrics(container.handle)
	return container.inner.Metrics()
	// return garden.Metrics{}, nil
}

func (container *VContainer) SetGraceTime(graceTime time.Duration) error {
	// return container.connection.SetGraceTime(container.handle, graceTime)
	return container.inner.SetGraceTime(graceTime)
	// return nil
}

func (container *VContainer) Properties() (garden.Properties, error) {
	//return container.connection.Properties(container.handle)
	// return garden.Properties{}, nil
	return container.inner.Properties()
}

func (container *VContainer) Property(name string) (string, error) {
	//return "", container.connection.Property(container.handle, name)
	// return "", nil
	return container.inner.Property(name)
}

func (container *VContainer) SetProperty(name string, value string) error {
	//return container.connection.SetProperty(container.handle, name, value)
	return container.inner.SetProperty(name, value)
}

func (container *VContainer) RemoveProperty(name string) error {
	//return container.connection.RemoveProperty(container.handle, name)
	// return nil
	return container.inner.RemoveProperty(name)
}
