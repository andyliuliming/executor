package vgarden // import "code.cloudfoundry.org/executor/depot/vci/vgarden"
import (
	"code.cloudfoundry.org/garden"
	GardenClient "code.cloudfoundry.org/garden/client"
	"code.cloudfoundry.org/lager"
)

type Client interface {
	garden.Client
}

type client struct {
	// connection connection.Connection
	inner  GardenClient.Client
	logger lager.Logger
}

func New() Client {
	return &client{
	// connection: connection,
	}
}

func NewWithAdapter(gc GardenClient.Client, logger lager.Logger) Client {
	return &client{
		inner:  gc,
		logger: logger,
	}
}

func (c *client) Ping() error {
	// return client.connection.Ping()
	return c.inner.Ping()
}

func (c *client) Capacity() (garden.Capacity, error) {
	// return client.connection.Capacity()

	// return garden.Capacity{}, nil

	c.logger.Info("########(andliu) Capacity")
	return c.inner.Capacity()
}

func (c *client) Create(spec garden.ContainerSpec) (garden.Container, error) {
	// handle, err := client.connection.Create(spec)
	// if err != nil {
	// 	return nil, err
	// }

	// return newContainer(handle, client.connection), nil

	c.logger.Info("########(andliu) Create", lager.Data{"spec": spec})
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
