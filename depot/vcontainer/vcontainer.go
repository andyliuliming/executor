package vcontainer

// maintain an adapter to the vcontainer rpc service.

import (
	"io"
	"time"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/lager"
	google_protobuf "github.com/gogo/protobuf/types"
	"github.com/virtualcloudfoundry/vcontainercommon"
	"github.com/virtualcloudfoundry/vcontainercommon/vcontainermodels"
	context "golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
)

type VContainer struct {
	handle           string
	inner            garden.Container
	vcontainerClient vcontainermodels.VContainerClient
	vprocessClient   vcontainermodels.VProcessClient
	logger           lager.Logger
}

func NewVContainerWithAdapter(inner garden.Container, logger lager.Logger, config vcontainermodels.VContainerClientConfig) garden.Container {
	vcontainerClient, err := NewVContainerClient(logger, config)
	if err != nil {
		logger.Error("new-vcontainer-client-failed", err)
	}
	vprocessClient, err := NewVProcessClient(logger, config)
	if err != nil {
		logger.Error("new-vprocess-client-failed", err)
	}
	return &VContainer{
		inner:            inner,
		vcontainerClient: vcontainerClient,
		vprocessClient:   vprocessClient,
		logger:           logger,
	}
}

func (c *VContainer) Handle() string {
	c.logger.Info("vcontainer-handle")
	return c.inner.Handle()
}

func (c *VContainer) Stop(kill bool) error {
	c.logger.Info("vcontainer-stop")

	ctx := c.buildContext()
	_, err := c.vcontainerClient.Stop(ctx, &vcontainermodels.StopMessage{
		Kill: kill,
	})

	if err != nil {
		c.logger.Error("vcontainer-stop", err, lager.Data{"kill": kill})
	}

	return c.inner.Stop(kill)
}

func (c *VContainer) Info() (garden.ContainerInfo, error) {
	c.logger.Info("vcontainer-info")
	ctx := c.buildContext()
	_, err := c.vcontainerClient.Info(ctx, &google_protobuf.Empty{})
	if err != nil {
		c.logger.Error("vcontainer-info", err)
	}
	return c.inner.Info()
}

func (c *VContainer) StreamIn(spec garden.StreamInSpec) error {
	c.logger.Info("vcontainer-stream-in")
	// TODO move share creation logic to here.
	ctx := c.buildContext()
	client, err := c.vcontainerClient.StreamIn(ctx)
	if err != nil {
		c.logger.Error("vcontainer-stream-in-failed", err)
		c.logger.Info("test-info-begin")
		_, err = c.Info()
		if err != nil {
			c.logger.Error("vcontainer-stream-in-test-info-failed", err)
		}
		c.logger.Info("test-info-end")
	} else {
		message := &vcontainermodels.StreamInSpec{
			Part: &vcontainermodels.StreamInSpec_Path{
				Path: spec.Path,
			},
		}

		err := client.Send(message)
		if err != nil {
			c.logger.Error("vcontainer-stream-in-spec-send-path-failed", err)
		}

		message = &vcontainermodels.StreamInSpec{
			Part: &vcontainermodels.StreamInSpec_User{
				User: spec.User,
			},
		}

		err = client.Send(message)
		if err != nil {
			c.logger.Error("vcontainer-stream-in-spec-send-user-failed", err)
		}

		// send the content now
		buf := make([]byte, 32*1024)
		bytesSent := 0
		for {
			nr, er := spec.TarStream.Read(buf)
			if er != nil {
				if er != io.EOF {
					err = er
				}
				break
			} else {
				if nr > 0 {
					bytesSent += nr
					message = &vcontainermodels.StreamInSpec{
						Part: &vcontainermodels.StreamInSpec_Content{
							Content: buf[:nr],
						},
					}
					c.logger.Info("vcontainer-stream-in-send-content-message-start")
					err = client.Send(message)
					c.logger.Info("vcontainer-stream-in-send-content-message-end")
					if err != nil {
						break
					}
				}
			}
		}
		if err != nil {
			c.logger.Error("vcontainer-stream-in-spec-send-content-failed", err)
		}
		streamInResponse, err := client.CloseAndRecv()
		if err != nil {
			c.logger.Error("vcontainer-stream-in-spec-close-and-recv-failed", err)
		}
		if streamInResponse != nil {
			c.logger.Info("vcontainer-stream-in-spec-close-and-recv", lager.Data{"message": streamInResponse.Message})
		}
		c.logger.Info("vcontainer-stream-in-spec-send-succeed", lager.Data{"bytes": bytesSent})
	}
	_, err = spec.TarStream.(io.Seeker).Seek(0, io.SeekStart)
	if err != nil {
		c.logger.Error("vcontainer-stream-seek-failed", err)
	}
	return c.inner.StreamIn(spec)
}

func (c *VContainer) StreamOut(spec garden.StreamOutSpec) (io.ReadCloser, error) {
	c.logger.Info("vcontainer-stream-out")
	ctx := c.buildContext()

	client, err := c.vcontainerClient.StreamOut(ctx, &vcontainermodels.StreamOutSpec{
		Path: spec.Path,
		User: spec.User,
	})

	if err != nil {
		c.logger.Error("vcontainer-stream-out", err)
	} else {
		stream := NewStreamOutAdapter(c.logger, client)
		data := make([]byte, 32*1024)
		for {
			n, err := stream.Read(data)
			if err != nil {
				c.logger.Error("vcontainer-stream-out-read-failed", err)
				break
			}
			c.logger.Info("read-got-byte-length", lager.Data{"n": n})
		}
	}
	return c.inner.StreamOut(spec)
}

func (c *VContainer) CurrentBandwidthLimits() (garden.BandwidthLimits, error) {
	c.logger.Info("vcontainer-current-bandwidth-limits")
	return c.inner.CurrentBandwidthLimits()
}

func (c *VContainer) CurrentCPULimits() (garden.CPULimits, error) {
	c.logger.Info("vcontainer-current-cpu-limits")
	return c.inner.CurrentCPULimits()
}

func (c *VContainer) CurrentDiskLimits() (garden.DiskLimits, error) {
	c.logger.Info("vcontainer-current-disk-limits")
	return c.inner.CurrentDiskLimits()
}

func (c *VContainer) CurrentMemoryLimits() (garden.MemoryLimits, error) {
	c.logger.Info("vcontainer-current-memory-limits")
	return c.inner.CurrentMemoryLimits()
}

func (c *VContainer) Run(spec garden.ProcessSpec, io garden.ProcessIO) (garden.Process, error) {
	c.logger.Info("vcontainer-run")
	ctx := c.buildContext()
	c.logger.Info("vcontainer-run-spec-origin", lager.Data{"spec": spec})
	specRemote, _ := ConvertProcessSpec(spec)
	c.logger.Info("vcontainer-run-spec-to-send", lager.Data{"spec": specRemote})
	runResponse, err := c.vcontainerClient.Run(ctx, specRemote)
	taskId := ""
	if err != nil {
		c.logger.Error("vcontainer-run", err)
		// return nil, err
	} else {
		c.logger.Info("vcontainer-run-spec-result-id", lager.Data{"id": runResponse.ID})
		taskId = runResponse.ID
	}

	innerProcess, err := c.inner.Run(spec, io)
	if err != nil {
		return innerProcess, err
	}
	process := NewVProcess(c.logger, c.inner.Handle(), taskId, innerProcess, c.vprocessClient)
	return process, nil
}

func (c *VContainer) Attach(processID string, io garden.ProcessIO) (garden.Process, error) {
	c.logger.Info("vcontainer-attach")
	return c.inner.Attach(processID, io)
}

func (c *VContainer) NetIn(hostPort, cPort uint32) (uint32, uint32, error) {
	c.logger.Info("vcontainer-net-in")
	return c.inner.NetIn(hostPort, cPort)
}

func (c *VContainer) NetOut(netOutRule garden.NetOutRule) error {
	c.logger.Info("vcontainer-net-out")
	return c.inner.NetOut(netOutRule)
}

func (c *VContainer) BulkNetOut(netOutRules []garden.NetOutRule) error {
	c.logger.Info("vcontainer-bulk-net-out")
	return c.inner.BulkNetOut(netOutRules)
}

func (c *VContainer) Metrics() (garden.Metrics, error) {
	c.logger.Info("vcontainer-metrics")
	ctx := c.buildContext()
	_, err := c.vcontainerClient.Metrics(ctx, &google_protobuf.Empty{})
	if err != nil {
		c.logger.Error("metrics", err)
	}
	return c.inner.Metrics()
}

func (c *VContainer) SetGraceTime(graceTime time.Duration) error {
	c.logger.Info("vcontainer-set-grace-time")
	protoDuration := &google_protobuf.Duration{}
	protoDuration.Seconds = int64(graceTime) / 1e9
	protoDuration.Nanos = int32(int64(graceTime) % 1e9)
	ctx := c.buildContext()
	_, err := c.vcontainerClient.SetGraceTime(ctx, protoDuration)
	if err != nil {
		c.logger.Error("set-grace-time", err)
	}
	return c.inner.SetGraceTime(graceTime)
}

func (c *VContainer) Properties() (garden.Properties, error) {
	c.logger.Info("vcontainer-properties")
	ctx := c.buildContext()
	_, err := c.vcontainerClient.Properties(ctx, &google_protobuf.Empty{})
	if err != nil {
		c.logger.Error("properties", err)
	}
	return c.inner.Properties()
}

func (c *VContainer) Property(name string) (string, error) {
	c.logger.Info("vcontainer-property")
	ctx := c.buildContext()
	_, err := c.vcontainerClient.Property(ctx, &google_protobuf.StringValue{
		Value: name,
	})
	if err != nil {
		c.logger.Error("property", err)
	}
	return c.inner.Property(name)
}

func (c *VContainer) SetProperty(name string, value string) error {
	c.logger.Info("vcontainer-set-property")
	ctx := c.buildContext()
	_, err := c.vcontainerClient.SetProperty(ctx, &vcontainermodels.KeyValueMessage{
		Key:   name,
		Value: value,
	})
	if err != nil {
		c.logger.Error("set-property", err)
	}
	return c.inner.SetProperty(name, value)
}

func (c *VContainer) RemoveProperty(name string) error {
	c.logger.Info("vcontainer-remove-property")
	ctx := c.buildContext()
	_, err := c.vcontainerClient.RemoveProperty(ctx, &google_protobuf.StringValue{
		Value: name,
	})
	if err != nil {
		c.logger.Error("remove-property", err)
	}
	return c.inner.RemoveProperty(name)
}

func (c *VContainer) buildContext() context.Context {
	md := metadata.Pairs(vcontainercommon.ContainerIDKey, c.Handle())
	ctx := context.Background()
	ctx = metadata.NewContext(ctx, md)
	return ctx
}
