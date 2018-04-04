package vcontainer

import google_protobuf3 "github.com/gogo/protobuf/types"
import (
	"context"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/lager"
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
	v.logger.Info("vprocess-wait")
	ctx := v.buildContext()
	client, err := v.vprocessClient.Wait(ctx)
	if err != nil {
		v.logger.Error("vprocess-wait-failed", err)
	}
	err = client.Send(&google_protobuf3.Empty{})
	if err != nil {
		v.logger.Error("vprocess-send-failed", err)
	}
	return v.inner.Wait()
}

func (v *VProcess) SetTTY(spec garden.TTYSpec) error {
	v.logger.Info("vprocess-set-tty")
	return v.inner.SetTTY(spec)
}

func (v *VProcess) Signal(sig garden.Signal) error {
	v.logger.Info("vprocess-signal")
	return v.inner.Signal(sig)
}

func (v *VProcess) buildContext() context.Context {
	md := metadata.Pairs(vcontainercommon.ContainerIDKey, v.containerID, vcontainercommon.ContainerIDKey, v.processID)
	ctx := context.Background()
	ctx = metadata.NewContext(ctx, md)
	return ctx
}
