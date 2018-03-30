package vcontainer

import (
	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/lager"
)

type VProcess struct {
	inner  garden.Process
	logger lager.Logger
}

func NewVProcess(logger lager.Logger, inner garden.Process) *VProcess {
	return &VProcess{
		inner:  inner,
		logger: logger,
	}
}

func (v *VProcess) ID() string {
	v.logger.Info("vprocess-id")
	return v.inner.ID()
}

func (v *VProcess) Wait() (int, error) {
	v.logger.Info("vprocess-wait")
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
