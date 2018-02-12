package containerstore

import (
	"os"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
)

type registryPruner struct {
	logger     lager.Logger
	config     *ContainerConfig
	clock      clock.Clock
	containers *nodeMap
}

func TempNewRegistryPruner(logger lager.Logger, config *ContainerConfig, clock clock.Clock, containers *nodeMap) *registryPruner {
	return &registryPruner{
		logger:     logger,
		config:     config,
		clock:      clock,
		containers: containers,
	}
}
func newRegistryPruner(logger lager.Logger, config *ContainerConfig, clock clock.Clock, containers *nodeMap) *registryPruner {
	return &registryPruner{
		logger:     logger,
		config:     config,
		clock:      clock,
		containers: containers,
	}
}

func (r *registryPruner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	logger := r.logger.Session("registry-pruner")
	ticker := r.clock.NewTicker(r.config.ReservedExpirationTime / 2)

	close(ready)

	defer ticker.Stop()
	for {
		select {
		case <-ticker.C():

			now := r.clock.Now()
			r.containers.CompleteExpired(logger, now)
		case signal := <-signals:
			logger.Info("signalled", lager.Data{"signal": signal.String()})
			return nil
		}
	}
}
