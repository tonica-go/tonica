package workflows

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// WorkflowInput represents the payload passed to Pace workflows.
type WorkflowInput struct {
	Workflow string            `json:"workflow"`
	Entity   string            `json:"entity"`
	RecordID string            `json:"record_id"`
	Payload  map[string]string `json:"payload"`
}

type WorkflowOutput struct {
	Result string `json:"result"`
}

type WorkerExample struct {
	temporalClient client.Client
}

// PaceWorkflow is a placeholder workflow that simply logs execution.
// It supports graceful cancellation by checking context at each step.
func (w *WorkerExample) PaceWorkflow(ctx workflow.Context, input WorkflowInput) (*WorkflowOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("pace workflow started",
		"workflow", input.Workflow,
		"entity", input.Entity,
		"record_id", input.RecordID,
		"payload", input.Payload,
	)

	// Check for cancellation at the start
	if err := ctx.Err(); err != nil {
		logger.Info("workflow cancelled before execution", "error", err)
		return nil, err
	}

	options := workflow.ActivityOptions{
		StartToCloseTimeout: 1 * time.Minute,
		HeartbeatTimeout:    15 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts:    3,
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, options)

	// Sleep with automatic cancellation check
	err := workflow.Sleep(ctx, time.Second)
	if err != nil {
		logger.Info("workflow cancelled during sleep", "error", err)
		return nil, err
	}

	result := ""
	//err = workflow.ExecuteActivity(ctx, w.DemoActivity, "Temporal").Get(ctx, &result)
	//if err != nil {
	//	// Check if this is a cancellation error
	//	if temporal.IsCanceledError(err) {
	//		logger.Info("workflow cancelled during first activity", "error", err)
	//		return nil, err
	//	}
	//	logger.Error("activity failed", "error", err)
	//	return nil, err
	//}

	result1 := ""
	err1 := workflow.ExecuteActivity(ctx, w.DemoActivity2, result).Get(ctx, &result1)
	if err1 != nil {
		// Check if this is a cancellation error
		if temporal.IsCanceledError(err1) {
			logger.Info("workflow cancelled during second activity", "error", err1)
			return nil, err1
		}
		logger.Error("activity failed", "error", err1)
		return nil, err1
	}

	logger.Info("pace workflow completed",
		"workflow", input.Workflow,
		"entity", input.Entity,
		"record_id", input.RecordID,
	)

	return &WorkflowOutput{Result: result1}, nil
}

// PaceWorkflowDummy is a placeholder workflow that simply logs execution.
// It supports graceful cancellation by checking context at each step.
func (w *WorkerExample) PaceWorkflowDummy(ctx workflow.Context, input WorkflowInput) (*WorkflowOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("pace workflow started",
		"workflow", input.Workflow,
		"entity", input.Entity,
		"record_id", input.RecordID,
		"payload", input.Payload,
	)

	// Check for cancellation at the start
	if err := ctx.Err(); err != nil {
		logger.Info("workflow cancelled before execution", "error", err)
		return nil, err
	}

	options := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
		// Enable heartbeat to allow activities to be cancelled mid-execution
		HeartbeatTimeout: 10 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, options)

	// Sleep with automatic cancellation check
	err := workflow.Sleep(ctx, time.Second)
	if err != nil {
		logger.Info("workflow cancelled during sleep", "error", err)
		return nil, err
	}
	return &WorkflowOutput{Result: "resp.Output"}, nil
}

func (w *WorkerExample) DemoActivity(ctx context.Context, name string) (string, error) {
	// Simulate long-running work with cancellation checks
	// Break the 5 second sleep into smaller chunks to check for cancellation
	for i := 0; i < 5; i++ {
		select {
		case <-ctx.Done():
			// Activity was cancelled, return immediately
			return "", fmt.Errorf("activity cancelled: %w", ctx.Err())
		case <-time.After(1 * time.Second):
			// Continue work
		}
	}
	return fmt.Sprintf("Hello from DemoActivity1, %s!", name), nil
}

func (w *WorkerExample) DemoActivity2(ctx context.Context, name string) (string, error) {
	// Quick check for cancellation even in fast activities
	activity.RecordHeartbeat(ctx, "step1")
	return fmt.Sprintf("Hello from DemoActivity2, %s!", name), nil
}
