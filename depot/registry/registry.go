package registry

import (
	"errors"
	"sync"

	"github.com/cloudfoundry-incubator/executor"
	"github.com/cloudfoundry/gunk/timeprovider"
	"github.com/tedsuo/ifrit"
)

var ErrContainerAlreadyExists = errors.New("container already exists")
var ErrContainerNotFound = errors.New("container not found")
var ErrContainerNotInitialized = errors.New("container not initialized")

var blankContainer = executor.Container{}

type Registry interface {
	CurrentCapacity() Capacity
	TotalCapacity() Capacity
	FindByGuid(guid string) (executor.Container, error)
	GetAllContainers() []executor.Container
	Reserve(guid string, req executor.ContainerAllocationRequest) (executor.Container, error)
	Initialize(guid string) (executor.Container, error)
	Create(guid, containerHandle string, ports []executor.PortMapping) (executor.Container, error)
	Start(guid string, process ifrit.Process) error
	Complete(guid string, result executor.ContainerRunResult) error
	Delete(guid string) error
	Sync(map[string]struct{})
}

type registry struct {
	totalCapacity        Capacity
	currentCapacity      *Capacity
	timeProvider         timeprovider.TimeProvider
	registeredContainers map[string]executor.Container
	containersMutex      *sync.RWMutex
}

func New(capacity Capacity, timeProvider timeprovider.TimeProvider) Registry {
	return &registry{
		totalCapacity:        capacity,
		currentCapacity:      &capacity,
		registeredContainers: make(map[string]executor.Container),
		containersMutex:      &sync.RWMutex{},
		timeProvider:         timeProvider,
	}
}

func (r *registry) TotalCapacity() Capacity {
	return r.totalCapacity
}

func (r *registry) CurrentCapacity() Capacity {
	r.containersMutex.RLock()
	defer r.containersMutex.RUnlock()

	return *r.currentCapacity
}

func (r *registry) GetAllContainers() []executor.Container {
	r.containersMutex.RLock()
	defer r.containersMutex.RUnlock()

	containers := []executor.Container{}
	for _, container := range r.registeredContainers {
		containers = append(containers, container)
	}

	return containers
}

func (r *registry) FindByGuid(guid string) (executor.Container, error) {
	r.containersMutex.RLock()
	defer r.containersMutex.RUnlock()

	res, ok := r.registeredContainers[guid]
	if !ok {
		return blankContainer, ErrContainerNotFound
	}

	return res, nil
}

func (r *registry) Reserve(guid string, req executor.ContainerAllocationRequest) (executor.Container, error) {
	res := executor.Container{
		Guid: guid,

		RootFSPath: req.RootFSPath,

		MemoryMB:   req.MemoryMB,
		DiskMB:     req.DiskMB,
		CpuPercent: req.CpuPercent,

		Ports: req.Ports,

		Log: req.Log,

		Tags: req.Tags,

		State:       executor.StateReserved,
		AllocatedAt: r.timeProvider.Time().UnixNano(),
	}

	r.containersMutex.Lock()
	defer r.containersMutex.Unlock()

	_, ok := r.registeredContainers[guid]
	if ok {
		return executor.Container{}, ErrContainerAlreadyExists
	}

	err := r.currentCapacity.alloc(res)
	if err != nil {
		return executor.Container{}, err
	}

	r.registeredContainers[res.Guid] = res

	return res, nil
}

func (r *registry) Initialize(guid string) (executor.Container, error) {
	r.containersMutex.Lock()
	defer r.containersMutex.Unlock()

	res, ok := r.registeredContainers[guid]
	if !ok {
		return blankContainer, ErrContainerNotFound
	}

	if res.State != executor.StateReserved {
		return blankContainer, ErrContainerNotInitialized
	}

	res.State = executor.StateInitializing

	r.registeredContainers[guid] = res
	return res, nil
}

func (r *registry) Create(guid, containerHandle string, ports []executor.PortMapping) (executor.Container, error) {
	r.containersMutex.Lock()
	defer r.containersMutex.Unlock()

	res, ok := r.registeredContainers[guid]
	if !ok {
		return blankContainer, ErrContainerNotFound
	}

	if res.State != executor.StateInitializing {
		return blankContainer, ErrContainerNotInitialized
	}

	res.State = executor.StateCreated
	res.ContainerHandle = containerHandle
	res.Ports = ports

	r.registeredContainers[guid] = res
	return res, nil
}

func (r *registry) Start(guid string, process ifrit.Process) error {
	r.containersMutex.Lock()
	defer r.containersMutex.Unlock()

	res, ok := r.registeredContainers[guid]
	if !ok {
		return ErrContainerNotFound
	}

	res.Process = process

	r.registeredContainers[guid] = res
	return nil
}

func (r *registry) Complete(guid string, result executor.ContainerRunResult) error {
	r.containersMutex.Lock()
	defer r.containersMutex.Unlock()

	res, ok := r.registeredContainers[guid]
	if !ok {
		return ErrContainerNotFound
	}

	res.State = executor.StateCompleted
	res.RunResult = result

	r.registeredContainers[guid] = res
	return nil
}

func (r *registry) Delete(guid string) error {
	r.containersMutex.Lock()
	defer r.containersMutex.Unlock()

	res, ok := r.registeredContainers[guid]
	if !ok {
		return ErrContainerNotFound
	}

	r.currentCapacity.free(res)
	delete(r.registeredContainers, guid)

	return nil
}

func (r *registry) Sync(handleSet map[string]struct{}) {
	r.containersMutex.Lock()
	defer r.containersMutex.Unlock()

	for guid, container := range r.registeredContainers {
		if container.ContainerHandle != "" {
			if _, ok := handleSet[container.ContainerHandle]; !ok {
				delete(r.registeredContainers, guid)
				r.currentCapacity.free(container)
			}
		}
	}
}