package execute_action

import (
	"github.com/cloudfoundry-incubator/executor/action_runner"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	steno "github.com/cloudfoundry/gosteno"
)

type ExecuteAction struct {
	runOnce *models.RunOnce
	logger  *steno.Logger
	action  action_runner.Action
}

func New(
	runOnce *models.RunOnce,
	logger *steno.Logger,
	action action_runner.Action,
) *ExecuteAction {
	return &ExecuteAction{
		runOnce: runOnce,
		logger:  logger,
		action:  action,
	}
}

func (action ExecuteAction) Perform() error {
	err := action.action.Perform()
	if err != nil {
		action.logger.Errord(
			map[string]interface{}{
				"runonce-guid": action.runOnce.Guid,
				"handle":       action.runOnce.ContainerHandle,
				"error":        err.Error(),
			},
			"runonce.actions.failed",
		)

		action.runOnce.Failed = true
		action.runOnce.FailureReason = err.Error()
	}

	return nil
}

func (action ExecuteAction) Cancel() {
	action.action.Cancel()
}

func (action ExecuteAction) Cleanup() {
	action.action.Cleanup()
}