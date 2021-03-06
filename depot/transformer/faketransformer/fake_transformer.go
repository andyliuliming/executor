// Code generated by counterfeiter. DO NOT EDIT.
package faketransformer

import (
	"sync"

	"code.cloudfoundry.org/executor"
	"code.cloudfoundry.org/executor/depot/log_streamer"
	"code.cloudfoundry.org/executor/depot/transformer"
	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/lager"
	"github.com/tedsuo/ifrit"
)

type FakeTransformer struct {
	StepsRunnerStub        func(lager.Logger, executor.Container, garden.Container, log_streamer.LogStreamer, transformer.Config) (ifrit.Runner, error)
	stepsRunnerMutex       sync.RWMutex
	stepsRunnerArgsForCall []struct {
		arg1 lager.Logger
		arg2 executor.Container
		arg3 garden.Container
		arg4 log_streamer.LogStreamer
		arg5 transformer.Config
	}
	stepsRunnerReturns struct {
		result1 ifrit.Runner
		result2 error
	}
	stepsRunnerReturnsOnCall map[int]struct {
		result1 ifrit.Runner
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeTransformer) StepsRunner(arg1 lager.Logger, arg2 executor.Container, arg3 garden.Container, arg4 log_streamer.LogStreamer, arg5 transformer.Config) (ifrit.Runner, error) {
	fake.stepsRunnerMutex.Lock()
	ret, specificReturn := fake.stepsRunnerReturnsOnCall[len(fake.stepsRunnerArgsForCall)]
	fake.stepsRunnerArgsForCall = append(fake.stepsRunnerArgsForCall, struct {
		arg1 lager.Logger
		arg2 executor.Container
		arg3 garden.Container
		arg4 log_streamer.LogStreamer
		arg5 transformer.Config
	}{arg1, arg2, arg3, arg4, arg5})
	fake.recordInvocation("StepsRunner", []interface{}{arg1, arg2, arg3, arg4, arg5})
	fake.stepsRunnerMutex.Unlock()
	if fake.StepsRunnerStub != nil {
		return fake.StepsRunnerStub(arg1, arg2, arg3, arg4, arg5)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.stepsRunnerReturns.result1, fake.stepsRunnerReturns.result2
}

func (fake *FakeTransformer) StepsRunnerCallCount() int {
	fake.stepsRunnerMutex.RLock()
	defer fake.stepsRunnerMutex.RUnlock()
	return len(fake.stepsRunnerArgsForCall)
}

func (fake *FakeTransformer) StepsRunnerArgsForCall(i int) (lager.Logger, executor.Container, garden.Container, log_streamer.LogStreamer, transformer.Config) {
	fake.stepsRunnerMutex.RLock()
	defer fake.stepsRunnerMutex.RUnlock()
	return fake.stepsRunnerArgsForCall[i].arg1, fake.stepsRunnerArgsForCall[i].arg2, fake.stepsRunnerArgsForCall[i].arg3, fake.stepsRunnerArgsForCall[i].arg4, fake.stepsRunnerArgsForCall[i].arg5
}

func (fake *FakeTransformer) StepsRunnerReturns(result1 ifrit.Runner, result2 error) {
	fake.StepsRunnerStub = nil
	fake.stepsRunnerReturns = struct {
		result1 ifrit.Runner
		result2 error
	}{result1, result2}
}

func (fake *FakeTransformer) StepsRunnerReturnsOnCall(i int, result1 ifrit.Runner, result2 error) {
	fake.StepsRunnerStub = nil
	if fake.stepsRunnerReturnsOnCall == nil {
		fake.stepsRunnerReturnsOnCall = make(map[int]struct {
			result1 ifrit.Runner
			result2 error
		})
	}
	fake.stepsRunnerReturnsOnCall[i] = struct {
		result1 ifrit.Runner
		result2 error
	}{result1, result2}
}

func (fake *FakeTransformer) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.stepsRunnerMutex.RLock()
	defer fake.stepsRunnerMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeTransformer) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ transformer.Transformer = new(FakeTransformer)
