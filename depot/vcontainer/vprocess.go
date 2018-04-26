package vcontainer

import (
	"context"
	"strings"
	"time"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/lager"
	google_protobuf "github.com/gogo/protobuf/types"
	"github.com/virtualcloudfoundry/vcontainercommon"
	"github.com/virtualcloudfoundry/vcontainercommon/vcontainermodels"
	"google.golang.org/grpc/metadata"
)

type VProcess struct {
	containerID    string
	processID      string
	inner          garden.Process
	logger         lager.Logger
	vprocessClient vcontainermodels.VProcessClient
}

func NewVProcess(logger lager.Logger, containerID, processID string, inner garden.Process,
	vprocessClient vcontainermodels.VProcessClient) *VProcess {
	logger.Info("vprocess-new-vprocess", lager.Data{"processID": processID, "containerID": containerID})
	return &VProcess{
		containerID:    containerID,
		processID:      processID,
		inner:          inner,
		logger:         logger,
		vprocessClient: vprocessClient,
	}
}

func (v *VProcess) ID() string {
	v.logger.Info("vprocess-id")
	return v.inner.ID()
}

func (v *VProcess) Wait() (int, error) {
	v.logger.Info("vprocess-wait", lager.Data{"processid": v.processID, "containerid": v.containerID})

	if len(v.containerID) != len("3fa79176-be9a-4496-bda2-cdaa06c32480") && // skip for the staging container for now.
		!strings.HasPrefix(v.containerID, "executor-healthcheck") {
		ctx := v.buildContext()
		client, err := v.vprocessClient.Wait(ctx, &google_protobuf.Empty{})
		if err != nil {
			v.logger.Error("vprocess-wait-failed", err)
		}
		defer func() {
			err = client.CloseSend()
			if err != nil {
				v.logger.Error("vprocess-wait-close-send-failed", err)
			}
		}()

		for {
			v.logger.Info("vprocess-wait-still-waiting", lager.Data{"processid": v.processID, "containerid": v.containerID})
			waitResponse, err := client.Recv()
			if err != nil {
				v.logger.Error("vprocess-wait-recv-failed", err, lager.Data{"processid": v.processID, "containerid": v.containerID})
				break
			}
			if waitResponse != nil && waitResponse.Exited {
				v.logger.Info("vprocess-wait-status-code", lager.Data{"status": waitResponse.ExitCode})
				break
			}
			// sleep for 5 seconds.
			time.Sleep(time.Second * 5)
		}
	}
	return v.inner.Wait()
}

func (v *VProcess) SetTTY(spec garden.TTYSpec) error {
	v.logger.Info("vprocess-set-tty", lager.Data{"processid": v.processID, "containerid": v.containerID})
	return v.inner.SetTTY(spec)
}

func (v *VProcess) Signal(sig garden.Signal) error {
	v.logger.Info("vprocess-signal", lager.Data{"processid": v.processID, "containerid": v.containerID})
	return v.inner.Signal(sig)
}

func (v *VProcess) buildContext() context.Context {
	md := metadata.Pairs(vcontainercommon.ContainerIDKey, v.containerID, vcontainercommon.ContainerIDKey, v.processID)
	ctx := context.Background()
	ctx = metadata.NewContext(ctx, md)
	return ctx
}
