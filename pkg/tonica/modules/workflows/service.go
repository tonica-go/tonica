package workflows

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
)

// TaskQueue is the Temporal task queue used by Pace workflows.
const TaskQueue = "default"

// Service coordinates workflow triggers via Temporal.
type Service struct {
	client        client.Client
	workerExample *WorkerExample
}

// NewService constructs Service.
func NewService(client client.Client) *Service {
	return &Service{
		client:        client,
		workerExample: &WorkerExample{temporalClient: client},
	}
}

// Trigger schedules a workflow execution in Temporal.
func (s *Service) Trigger(ctx context.Context, workflow string, entity string, recordID string, input map[string]string) (string, string, error) {
	if s.client == nil {
		return "", "", fmt.Errorf("temporal client unavailable")
	}

	if strings.TrimSpace(workflow) == "" {
		return "", "", fmt.Errorf("workflow name is required")
	}

	wfInput := WorkflowInput{
		Workflow: workflow,
		Entity:   entity,
		RecordID: recordID,
		Payload:  input,
	}

	workflowID := fmt.Sprintf("%s-%s-%s-%s", entity, recordID, workflow, uuid.NewString())
	options := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: TaskQueue,
	}

	run, err := s.client.ExecuteWorkflow(ctx, options, workflow, wfInput)
	if err != nil {
		return "", "", err
	}

	result := "unknown"
	var resultObject WorkflowOutput
	if err := run.Get(ctx, &resultObject); err != nil {
		return run.GetID(), "failed", err
	}

	if resultObject.Result == "" {
		result = "completed"
	}

	return run.GetID(), result, nil
}
