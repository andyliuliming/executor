package delete_container

import (
	"net/http"
	"sync"

	"github.com/cloudfoundry-incubator/executor/registry"
	"github.com/cloudfoundry-incubator/garden/warden"
	"github.com/cloudfoundry/gosteno"
)

type handler struct {
	wardenClient warden.Client
	registry     registry.Registry
	waitGroup    *sync.WaitGroup
	logger       *gosteno.Logger
}

func New(wardenClient warden.Client, registry registry.Registry, waitGroup *sync.WaitGroup, logger *gosteno.Logger) http.Handler {
	return &handler{
		wardenClient: wardenClient,
		registry:     registry,
		waitGroup:    waitGroup,
		logger:       logger,
	}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.waitGroup.Add(1)
	defer h.waitGroup.Done()

	guid := r.FormValue(":guid")

	container, err := h.registry.FindByGuid(guid)
	if err != nil {
		handleError(err, w, h.logger)
		return
	}

	//TODO once wardenClient has an ErrNotFound error code, use it
	//to determine if we should delete from registry
	if container.ContainerHandle != "" {
		h.wardenClient.Destroy(container.ContainerHandle)
	}

	err = h.registry.Delete(guid)
	if err != nil {
		handleError(err, w, h.logger)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleError(err error, w http.ResponseWriter, logger *gosteno.Logger) {
	if err == registry.ErrContainerNotFound {
		logger.Infod(map[string]interface{}{
			"error": err.Error(),
		}, "executor.delete-container.not-found")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	logger.Errord(map[string]interface{}{
		"error": err.Error(),
	}, "executor.delete-container.failed")
	w.WriteHeader(http.StatusInternalServerError)
}