package vcontainer

import (
	"code.cloudfoundry.org/garden"
	google_protobuf "github.com/gogo/protobuf/types"
	"github.com/virtualcloudfoundry/vcontainercommon/vcontainermodels"
)

func ConvertProcessSpec(spec garden.ProcessSpec) (*vcontainermodels.ProcessSpec, error) {
	converted := vcontainermodels.ProcessSpec{
		ID:   spec.ID,
		Args: spec.Args,
		Path: spec.Path,
		Env:  spec.Env,
		Dir:  spec.Dir,
		User: spec.User,
	}
	for _, bindMount := range spec.BindMounts {
		converted.BindMounts = append(converted.BindMounts, vcontainermodels.BindMount{
			SrcPath: bindMount.SrcPath,
			DstPath: bindMount.DstPath,
			Mode:    vcontainermodels.BindMountMode(bindMount.Mode),
			Origin:  vcontainermodels.BindMountOrigin(bindMount.Origin),
		})
	}
	return &converted, nil
}

func ConvertContainerSpec(spec garden.ContainerSpec) (*vcontainermodels.ContainerSpec, error) {
	protoDuration := &google_protobuf.Duration{}
	protoDuration.Seconds = int64(spec.GraceTime) / 1e9
	protoDuration.Nanos = int32(int64(spec.GraceTime) % 1e9)

	converted := vcontainermodels.ContainerSpec{
		Handle:     spec.Handle,
		GraceTime:  protoDuration,
		RootFSPath: spec.RootFSPath,
		Image: vcontainermodels.ImageRef{
			URI:      spec.Image.URI,
			Username: spec.Image.Username,
			Password: spec.Image.Password,
		},
		Network:    spec.Network,
		Env:        spec.Env,
		Privileged: spec.Privileged,
	}

	for _, bindMount := range spec.BindMounts {
		converted.BindMounts = append(converted.BindMounts, vcontainermodels.BindMount{
			SrcPath: bindMount.SrcPath,
			DstPath: bindMount.DstPath,
			Mode:    vcontainermodels.BindMountMode(bindMount.Mode),
			Origin:  vcontainermodels.BindMountOrigin(bindMount.Origin),
		})
	}

	for _, netIn := range spec.NetIn {
		converted.NetIn = append(converted.NetIn, vcontainermodels.NetInRequest{
			HostPort:      netIn.HostPort,
			ContainerPort: netIn.ContainerPort,
		})
	}

	for _, netOut := range spec.NetOut {
		converted.NetOut = append(converted.NetOut, ConvertNetOutRule(&netOut))
	}

	// TODO copy the limits.
	return &converted, nil
}

func ConvertNetOutRule(origin *garden.NetOutRule) vcontainermodels.NetOutRuleRequest {
	return vcontainermodels.NetOutRuleRequest{
		Log: origin.Log,
	}
}

func ConvertProperties() {

}

func ConvertBackContainerInfo() {

}
