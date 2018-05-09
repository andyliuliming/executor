package vcontainer

// maintain an adapter to the vcontainer  rpc service.
import (
	"context"

	"github.com/virtualcloudfoundry/vcontainercommon/vcontainermodels"

	"code.cloudfoundry.org/garden"
	GardenClient "code.cloudfoundry.org/garden/client"
	"code.cloudfoundry.org/lager"
	google_protobuf "github.com/gogo/protobuf/types"
	vcmodels "github.com/virtualcloudfoundry/vcontainercommon/vcontainermodels"
)

type vgarden struct {
	inner         GardenClient.Client
	config        vcmodels.VContainerClientConfig
	vgardenClient vcmodels.VGardenClient
	logger        lager.Logger
}

func NewVGardenWithAdapter(gc GardenClient.Client, logger lager.Logger, config vcmodels.VContainerClientConfig) garden.Client {
	vgardenClient, err := NewVGardenClient(logger, config)
	if err != nil {
		logger.Error("new-vgarden-with-adapter", err)
	}
	return &vgarden{
		inner:         gc,
		config:        config,
		vgardenClient: vgardenClient,
		logger:        logger,
	}
}

func (c *vgarden) Ping() error {
	c.logger.Info("vgarden-ping")
	_, err := c.vgardenClient.Ping(context.Background(), &google_protobuf.Empty{})
	if err != nil {
		c.logger.Error("ping", err)
	}
	return c.inner.Ping()
}

func (c *vgarden) Capacity() (garden.Capacity, error) {
	c.logger.Info("vgarden-capacity")
	_, err := c.vgardenClient.Capacity(context.Background(), &google_protobuf.Empty{})
	if err != nil {
		c.logger.Error("ping", err)
	}
	return c.inner.Capacity()
}

func (c *vgarden) Create(spec garden.ContainerSpec) (garden.Container, error) {
	c.logger.Info("vgarden-create")
	c.logger.Info("vgarden-create-container-handle", lager.Data{"handle": spec.Handle})

	specRemote, _ := ConvertContainerSpec(spec)
	_, err := c.vgardenClient.Create(context.Background(), specRemote)
	if err != nil {
		c.logger.Error("create", err)
	}
	return c.inner.Create(spec)
}

func (c *vgarden) Containers(properties garden.Properties) ([]garden.Container, error) {
	// we only support get all containers.
	c.logger.Info("vgarden-containers", lager.Data{"properties": properties})

	_, err := c.vgardenClient.Containers(context.Background(), &vcontainermodels.Properties{})
	if err != nil {
		c.logger.Error("containers", err)
	}
	return c.inner.Containers(properties)
}

func (c *vgarden) Destroy(handle string) error {
	c.logger.Info("vgarden-destroy", lager.Data{"handle": handle})

	_, err := c.vgardenClient.Destroy(context.Background(), &google_protobuf.StringValue{
		Value: handle,
	})
	if err != nil {
		c.logger.Error("destroy", err)
	}
	return c.inner.Destroy(handle)
}

func (c *vgarden) BulkInfo(handles []string) (map[string]garden.ContainerInfoEntry, error) {
	c.logger.Info("vgarden-bulkinfo")
	_, err := c.vgardenClient.BulkInfo(context.Background(), &vcontainermodels.BulkInfoRequest{
		Handles: handles,
	})
	if err != nil {
		c.logger.Error("bulkinfo", err)
	}
	return c.inner.BulkInfo(handles)
}

func (c *vgarden) BulkMetrics(handles []string) (map[string]garden.ContainerMetricsEntry, error) {
	c.logger.Info("vgarden-bulkmetrics")
	_, err := c.vgardenClient.BulkMetrics(context.Background(), &vcontainermodels.BulkMetricsRequest{
		Handles: handles,
	})
	if err != nil {
		c.logger.Error("bulkmetrics", err)
	}
	bulkMetrics, err := c.inner.BulkMetrics(handles)
	// return c.inner.BulkMetrics(handles)
	c.logger.Info("vgarden-bulkmetrics-real-data", lager.Data{"metrics": bulkMetrics})
	return bulkMetrics, err
}

func (c *vgarden) Lookup(handle string) (garden.Container, error) {
	c.logger.Info("vgarden-lookup")
	_, err := c.vgardenClient.Lookup(context.Background(), &google_protobuf.StringValue{
		Value: handle,
	})
	if err != nil {
		c.logger.Error("lookup", err)
	}
	return c.inner.Lookup(handle)
}
