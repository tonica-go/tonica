package workflows

import (
	"context"
	"fmt"

	pacev1 "github.com/tonica-go/tonica/pkg/tonica/proto/workflows"

	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/failure/v1"
	"go.temporal.io/api/schedule/v1"
	"go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Temporary stubs - these will be properly implemented later

func (s *Service) ListNamespaces(ctx context.Context) ([]*pacev1.Namespace, error) {
	res, err := s.client.WorkflowService().ListNamespaces(ctx, &workflowservice.ListNamespacesRequest{
		PageSize: 100,
	})
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}

	var namespaces []*pacev1.Namespace
	for _, ns := range res.Namespaces {
		namespaces = append(namespaces, &pacev1.Namespace{
			Name:        ns.NamespaceInfo.Name,
			Description: ns.NamespaceInfo.Description,
		})
	}

	return namespaces, nil
}

func (s *Service) ListWorkflows(ctx context.Context, namespace string, requestedWorkflowType string, status pacev1.WorkflowStatus, pageSize int32, pageToken string, searchQuery string) ([]*pacev1.WorkflowExecution, string, error) {
	if searchQuery != "" {
		searchQuery = fmt.Sprintf("`WorkflowId` STARTS_WITH \"%s\"", searchQuery)
	}
	res, err := s.client.WorkflowService().ListWorkflowExecutions(ctx, &workflowservice.ListWorkflowExecutionsRequest{
		Namespace:     namespace,
		PageSize:      pageSize,
		Query:         searchQuery,
		NextPageToken: []byte(pageToken),
	})
	if err != nil {
		return nil, "", fmt.Errorf("list workflows: %w", err)
	}
	var workflows []*pacev1.WorkflowExecution
	for _, we := range res.Executions {
		workflowStatus := pacev1.WorkflowStatus_WORKFLOW_STATUS_UNSPECIFIED
		switch we.Status {
		case 0: // WORKFLOW_EXECUTION_STATUS_UNSPECIFIED
			workflowStatus = pacev1.WorkflowStatus_WORKFLOW_STATUS_UNSPECIFIED
		case 1: // WORKFLOW_EXECUTION_STATUS_RUNNING
			workflowStatus = pacev1.WorkflowStatus_WORKFLOW_STATUS_RUNNING
		case 2: // WORKFLOW_EXECUTION_STATUS_COMPLETED
			workflowStatus = pacev1.WorkflowStatus_WORKFLOW_STATUS_COMPLETED
		case 3: // WORKFLOW_EXECUTION_STATUS_FAILED
			workflowStatus = pacev1.WorkflowStatus_WORKFLOW_STATUS_FAILED
		case 4: // WORKFLOW_EXECUTION_STATUS_CANCELED
			workflowStatus = pacev1.WorkflowStatus_WORKFLOW_STATUS_CANCELED
		case 5: // WORKFLOW_EXECUTION_STATUS_TERMINATED
			workflowStatus = pacev1.WorkflowStatus_WORKFLOW_STATUS_TERMINATED
		case 6: // WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW
			workflowStatus = pacev1.WorkflowStatus_WORKFLOW_STATUS_CONTINUED_AS_NEW
		case 7: // WORKFLOW_EXECUTION_STATUS_TIMED_OUT
			workflowStatus = pacev1.WorkflowStatus_WORKFLOW_STATUS_TIMED_OUT
		}

		workflowType := we.Type.Name
		if workflowType == "" {
			workflowType = "unknown"
		}
		if requestedWorkflowType != "" && workflowType != requestedWorkflowType {
			continue
		}

		if status != pacev1.WorkflowStatus_WORKFLOW_STATUS_UNSPECIFIED && workflowStatus != status {
			continue
		}
		workflows = append(workflows, &pacev1.WorkflowExecution{
			WorkflowId:       we.Execution.WorkflowId,
			RunId:            we.Execution.RunId,
			Status:           workflowStatus,
			WorkflowType:     workflowType,
			StartTime:        we.StartTime,
			CloseTime:        we.CloseTime,
			HistoryLength:    we.HistoryLength,
			ParentWorkflowId: we.ParentExecution.GetWorkflowId(),
			ParentRunId:      we.ParentExecution.GetRunId(),
			//SearchAttributes: we.SearchAttributes,
		})
	}

	return workflows, "", nil
}

func (s *Service) GetWorkflow(ctx context.Context, namespace string, workflowID string, runID string) (*pacev1.WorkflowDetails, error) {
	req := &workflowservice.DescribeWorkflowExecutionRequest{
		Namespace: namespace,
		Execution: &common.WorkflowExecution{
			WorkflowId: workflowID,
			RunId:      runID,
		},
	}

	resp, err := s.client.WorkflowService().DescribeWorkflowExecution(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("describe workflow: %w", err)
	}

	info := resp.WorkflowExecutionInfo

	// Map status
	workflowStatus := pacev1.WorkflowStatus_WORKFLOW_STATUS_UNSPECIFIED
	switch info.Status {
	case 1:
		workflowStatus = pacev1.WorkflowStatus_WORKFLOW_STATUS_RUNNING
	case 2:
		workflowStatus = pacev1.WorkflowStatus_WORKFLOW_STATUS_COMPLETED
	case 3:
		workflowStatus = pacev1.WorkflowStatus_WORKFLOW_STATUS_FAILED
	case 4:
		workflowStatus = pacev1.WorkflowStatus_WORKFLOW_STATUS_CANCELED
	case 5:
		workflowStatus = pacev1.WorkflowStatus_WORKFLOW_STATUS_TERMINATED
	case 6:
		workflowStatus = pacev1.WorkflowStatus_WORKFLOW_STATUS_CONTINUED_AS_NEW
	case 7:
		workflowStatus = pacev1.WorkflowStatus_WORKFLOW_STATUS_TIMED_OUT
	}

	execution := &pacev1.WorkflowExecution{
		WorkflowId:    info.Execution.WorkflowId,
		RunId:         info.Execution.RunId,
		WorkflowType:  info.Type.Name,
		Namespace:     namespace,
		Status:        workflowStatus,
		StartTime:     info.StartTime,
		CloseTime:     info.CloseTime,
		HistoryLength: info.HistoryLength,
	}

	// Parse pending activities
	var pendingActivities []*pacev1.PendingActivity
	for _, pa := range resp.PendingActivities {
		pendingActivities = append(pendingActivities, &pacev1.PendingActivity{
			ActivityId:        pa.ActivityId,
			ActivityType:      pa.ActivityType.Name,
			ScheduledTime:     pa.ScheduledTime,
			LastHeartbeatTime: pa.LastHeartbeatTime,
			Attempt:           pa.Attempt,
		})
	}

	// Get full f information if exists
	var f *structpb.Struct

	// Get workflow input and result from history
	var input, result *structpb.Struct

	// Fetch ALL history events to get input from first event and result from last event
	// We need to paginate through all events to find the actual last event
	historyReq := &workflowservice.GetWorkflowExecutionHistoryRequest{
		Namespace: namespace,
		Execution: &common.WorkflowExecution{
			WorkflowId: workflowID,
			RunId:      runID,
		},
		MaximumPageSize: 1000, // Get all events in one request (most workflows have < 1000 events)
	}

	historyResp, err := s.client.WorkflowService().GetWorkflowExecutionHistory(ctx, historyReq)
	if err == nil && historyResp.History != nil && len(historyResp.History.Events) > 0 {
		allEvents := historyResp.History.Events

		// If there are more pages, fetch them all
		for len(historyResp.NextPageToken) > 0 {
			historyReq.NextPageToken = historyResp.NextPageToken
			historyResp, err = s.client.WorkflowService().GetWorkflowExecutionHistory(ctx, historyReq)
			if err != nil {
				break
			}
			if historyResp.History != nil {
				allEvents = append(allEvents, historyResp.History.Events...)
			}
		}

		// Get input from first event
		firstEvent := allEvents[0]
		if firstEvent.GetWorkflowExecutionStartedEventAttributes() != nil {
			startAttrs := firstEvent.GetWorkflowExecutionStartedEventAttributes()
			if startAttrs.Input != nil && len(startAttrs.Input.Payloads) > 0 {
				input, _ = payloadToStruct(startAttrs.Input.Payloads[0])
			}
		}

		// Get result from last event
		lastEvent := allEvents[len(allEvents)-1]
		if info.Status == 2 { // COMPLETED
			if lastEvent.GetWorkflowExecutionCompletedEventAttributes() != nil {
				completeAttrs := lastEvent.GetWorkflowExecutionCompletedEventAttributes()
				if completeAttrs.Result != nil && len(completeAttrs.Result.Payloads) > 0 {
					result, _ = payloadToStruct(completeAttrs.Result.Payloads[0])
				}
			}
		}

		// Get full f information from WorkflowExecutionFailed event
		if info.Status == 3 { // FAILED
			if lastEvent.GetWorkflowExecutionFailedEventAttributes() != nil {
				failedAttrs := lastEvent.GetWorkflowExecutionFailedEventAttributes()
				if failedAttrs.Failure != nil {
					f = failureToStruct(failedAttrs.Failure)
				}
			}
		}
	}

	// Initialize empty structs if nil
	if input == nil {
		input = &structpb.Struct{}
	}
	if result == nil {
		result = &structpb.Struct{}
	}

	details := &pacev1.WorkflowDetails{
		Execution:         execution,
		PendingActivities: pendingActivities,
		FailureMessage:    f,
		ExecutionTime:     info.ExecutionTime,
		Input:             input,
		Result:            result,
	}

	return details, nil
}

func failureToStruct(failure *failure.Failure) *structpb.Struct {
	if failure == nil {
		return &structpb.Struct{}
	}

	fields := map[string]*structpb.Value{
		"message":             structpb.NewStringValue(failure.GetMessage()),
		"source":              structpb.NewStringValue(failure.GetSource()),
		"stackTrace":          structpb.NewStringValue(failure.GetStackTrace()),
		"cause":               structpb.NewStringValue(failure.GetCause().String()),
		"attributes":          structpb.NewStringValue(failure.GetEncodedAttributes().String()),
		"activityFailureInfo": activityFailureToStruct(failure.GetActivityFailureInfo()),
	}

	return &structpb.Struct{
		Fields: fields,
	}
}

func activityFailureToStruct(info *failure.ActivityFailureInfo) *structpb.Value {
	if info == nil {
		return structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{}}) // Empty struct
	}

	return structpb.NewStructValue(&structpb.Struct{
		Fields: map[string]*structpb.Value{
			"scheduledEventId": structpb.NewNumberValue(float64(info.ScheduledEventId)),
			"startedEventId":   structpb.NewNumberValue(float64(info.StartedEventId)),
			"identity":         structpb.NewStringValue(info.Identity),
			"activityType":     structpb.NewStringValue(info.ActivityType.String()),
			"activityId":       structpb.NewStringValue(info.ActivityId),
			"retryState":       structpb.NewStringValue(info.RetryState.String()),
		},
	})
}

func payloadToStruct(payload *common.Payload) (*structpb.Struct, error) {
	if payload == nil || payload.GetData() == nil {
		return &structpb.Struct{}, nil
	}

	data := payload.GetData()

	// Check encoding metadata
	encoding := "json/plain"
	if payload.Metadata != nil {
		if enc, ok := payload.Metadata["encoding"]; ok {
			encoding = string(enc)
		}
	}

	// Handle JSON encoding (most common for Temporal)
	if encoding == "json/plain" || encoding == "" {
		// First, try to unmarshal as a generic Value (handles primitives and objects)
		var value structpb.Value
		if err := value.UnmarshalJSON(data); err == nil {
			// If it's a struct value, return it directly
			if structVal := value.GetStructValue(); structVal != nil {
				return structVal, nil
			}
			// Otherwise, wrap primitive value in a struct with "value" field
			return &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"value": &value,
				},
			}, nil
		}

		// If that fails, try to unmarshal as Struct directly
		var s structpb.Struct
		if err := s.UnmarshalJSON(data); err != nil {
			// If unmarshal fails, try to wrap raw data as a string value
			return &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"data": structpb.NewStringValue(string(data)),
				},
			}, nil
		}
		return &s, nil
	}

	// Fallback: try protobuf unmarshal
	var s structpb.Struct
	if err := proto.Unmarshal(data, &s); err != nil {
		// Last resort: return raw data as string
		return &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"raw": structpb.NewStringValue(string(data)),
			},
		}, nil
	}

	return &s, nil
}

func (s *Service) GetWorkflowHistory(ctx context.Context, namespace string, workflowID string, runID string, pageSize int32, pageToken string) ([]*pacev1.HistoryEvent, string, error) {
	if pageSize == 0 {
		pageSize = 100
	}

	req := &workflowservice.GetWorkflowExecutionHistoryRequest{
		Namespace: namespace,
		Execution: &common.WorkflowExecution{
			WorkflowId: workflowID,
			RunId:      runID,
		},
		MaximumPageSize: pageSize,
		NextPageToken:   []byte(pageToken),
	}

	resp, err := s.client.WorkflowService().GetWorkflowExecutionHistory(ctx, req)
	if err != nil {
		return nil, "", fmt.Errorf("get workflow history: %w", err)
	}

	var events []*pacev1.HistoryEvent
	for _, he := range resp.History.Events {
		// Map event type
		eventType := pacev1.HistoryEventType_HISTORY_EVENT_TYPE_UNSPECIFIED
		eventName := ""
		var eventAttrs *structpb.Struct

		switch he.EventType {
		case enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED:
			eventType = pacev1.HistoryEventType_HISTORY_EVENT_TYPE_WORKFLOW_EXECUTION_STARTED
		case enums.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED:
			eventType = pacev1.HistoryEventType_HISTORY_EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED
		case enums.EVENT_TYPE_WORKFLOW_EXECUTION_FAILED:
			eventType = pacev1.HistoryEventType_HISTORY_EVENT_TYPE_WORKFLOW_EXECUTION_FAILED
		case enums.EVENT_TYPE_WORKFLOW_EXECUTION_TIMED_OUT:
			eventType = pacev1.HistoryEventType_HISTORY_EVENT_TYPE_WORKFLOW_EXECUTION_TIMED_OUT
		case enums.EVENT_TYPE_WORKFLOW_EXECUTION_CANCELED:
			eventType = pacev1.HistoryEventType_HISTORY_EVENT_TYPE_WORKFLOW_EXECUTION_CANCELED
		case enums.EVENT_TYPE_WORKFLOW_EXECUTION_TERMINATED:
			eventType = pacev1.HistoryEventType_HISTORY_EVENT_TYPE_WORKFLOW_EXECUTION_TERMINATED
		case enums.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED:
			eventType = pacev1.HistoryEventType_HISTORY_EVENT_TYPE_WORKFLOW_TASK_SCHEDULED
			// No attributes needed for scheduled event
		case enums.EVENT_TYPE_WORKFLOW_TASK_STARTED:
			eventType = pacev1.HistoryEventType_HISTORY_EVENT_TYPE_WORKFLOW_TASK_STARTED
			if attrs := he.GetWorkflowTaskStartedEventAttributes(); attrs != nil {
				eventAttrs = &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"scheduledEventId": structpb.NewNumberValue(float64(attrs.ScheduledEventId)),
					},
				}
			}
		case enums.EVENT_TYPE_WORKFLOW_TASK_COMPLETED:
			eventType = pacev1.HistoryEventType_HISTORY_EVENT_TYPE_WORKFLOW_TASK_COMPLETED
			if attrs := he.GetWorkflowTaskCompletedEventAttributes(); attrs != nil {
				eventAttrs = &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"scheduledEventId": structpb.NewNumberValue(float64(attrs.ScheduledEventId)),
						"startedEventId":   structpb.NewNumberValue(float64(attrs.StartedEventId)),
					},
				}
			}
		case enums.EVENT_TYPE_WORKFLOW_TASK_FAILED:
			eventType = pacev1.HistoryEventType_HISTORY_EVENT_TYPE_WORKFLOW_TASK_FAILED
			if attrs := he.GetWorkflowTaskFailedEventAttributes(); attrs != nil {
				eventAttrs = &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"scheduledEventId": structpb.NewNumberValue(float64(attrs.ScheduledEventId)),
						"startedEventId":   structpb.NewNumberValue(float64(attrs.StartedEventId)),
					},
				}
			}
		case enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED:
			eventType = pacev1.HistoryEventType_HISTORY_EVENT_TYPE_ACTIVITY_TASK_SCHEDULED
			if attrs := he.GetActivityTaskScheduledEventAttributes(); attrs != nil {
				if attrs.ActivityType != nil {
					eventName = attrs.ActivityType.Name
				}
				attrsData := &structpb.Struct{
					Fields: make(map[string]*structpb.Value),
				}
				attrsData.Fields["activityId"] = structpb.NewStringValue(attrs.ActivityId)

				if attrs.Input != nil && len(attrs.Input.Payloads) > 0 {
					inputStruct, err := payloadToStruct(attrs.Input.Payloads[0])
					if err == nil {
						attrsData.Fields["input"] = structpb.NewStructValue(inputStruct)
					}
				}
				eventAttrs = attrsData
			}
		case enums.EVENT_TYPE_ACTIVITY_TASK_STARTED:
			eventType = pacev1.HistoryEventType_HISTORY_EVENT_TYPE_ACTIVITY_TASK_STARTED
			if attrs := he.GetActivityTaskStartedEventAttributes(); attrs != nil {
				eventAttrs = &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"scheduledEventId": structpb.NewNumberValue(float64(attrs.ScheduledEventId)),
					},
				}
			}
		case enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED:
			eventType = pacev1.HistoryEventType_HISTORY_EVENT_TYPE_ACTIVITY_TASK_COMPLETED
			if attrs := he.GetActivityTaskCompletedEventAttributes(); attrs != nil {
				attrsData := &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"scheduledEventId": structpb.NewNumberValue(float64(attrs.ScheduledEventId)),
						"startedEventId":   structpb.NewNumberValue(float64(attrs.StartedEventId)),
					},
				}
				if attrs.Result != nil && len(attrs.Result.Payloads) > 0 {
					resultStruct, err := payloadToStruct(attrs.Result.Payloads[0])
					if err == nil {
						attrsData.Fields["result"] = structpb.NewStructValue(resultStruct)
					}
				}
				eventAttrs = attrsData
			}
		case enums.EVENT_TYPE_ACTIVITY_TASK_FAILED:
			eventType = pacev1.HistoryEventType_HISTORY_EVENT_TYPE_ACTIVITY_TASK_FAILED
			if attrs := he.GetActivityTaskFailedEventAttributes(); attrs != nil {
				eventAttrs = &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"scheduledEventId": structpb.NewNumberValue(float64(attrs.ScheduledEventId)),
						"startedEventId":   structpb.NewNumberValue(float64(attrs.StartedEventId)),
					},
				}
			}
		case enums.EVENT_TYPE_ACTIVITY_TASK_TIMED_OUT:
			eventType = pacev1.HistoryEventType_HISTORY_EVENT_TYPE_ACTIVITY_TASK_TIMED_OUT
			if attrs := he.GetActivityTaskTimedOutEventAttributes(); attrs != nil {
				eventAttrs = &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"scheduledEventId": structpb.NewNumberValue(float64(attrs.ScheduledEventId)),
						"startedEventId":   structpb.NewNumberValue(float64(attrs.StartedEventId)),
					},
				}
			}
		case enums.EVENT_TYPE_ACTIVITY_TASK_CANCEL_REQUESTED:
			eventType = pacev1.HistoryEventType_HISTORY_EVENT_TYPE_ACTIVITY_TASK_CANCEL_REQUESTED
		case enums.EVENT_TYPE_ACTIVITY_TASK_CANCELED:
			eventType = pacev1.HistoryEventType_HISTORY_EVENT_TYPE_ACTIVITY_TASK_CANCELED
			if attrs := he.GetActivityTaskCanceledEventAttributes(); attrs != nil {
				eventAttrs = &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"scheduledEventId": structpb.NewNumberValue(float64(attrs.ScheduledEventId)),
						"startedEventId":   structpb.NewNumberValue(float64(attrs.StartedEventId)),
					},
				}
			}
		case enums.EVENT_TYPE_TIMER_STARTED:
			eventType = pacev1.HistoryEventType_HISTORY_EVENT_TYPE_TIMER_STARTED
			if attrs := he.GetTimerStartedEventAttributes(); attrs != nil {
				eventName = "Timer: " + attrs.TimerId
				eventAttrs = &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"timerId": structpb.NewStringValue(attrs.TimerId),
					},
				}
			}
		case enums.EVENT_TYPE_TIMER_FIRED:
			eventType = pacev1.HistoryEventType_HISTORY_EVENT_TYPE_TIMER_FIRED
			if attrs := he.GetTimerFiredEventAttributes(); attrs != nil {
				eventName = "Timer: " + attrs.TimerId
				eventAttrs = &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"timerId":        structpb.NewStringValue(attrs.TimerId),
						"startedEventId": structpb.NewNumberValue(float64(attrs.StartedEventId)),
					},
				}
			}
		case enums.EVENT_TYPE_TIMER_CANCELED:
			eventType = pacev1.HistoryEventType_HISTORY_EVENT_TYPE_TIMER_CANCELED
			if attrs := he.GetTimerCanceledEventAttributes(); attrs != nil {
				eventName = "Timer: " + attrs.TimerId
				eventAttrs = &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"timerId":        structpb.NewStringValue(attrs.TimerId),
						"startedEventId": structpb.NewNumberValue(float64(attrs.StartedEventId)),
					},
				}
			}
		case enums.EVENT_TYPE_MARKER_RECORDED:
			eventType = pacev1.HistoryEventType_HISTORY_EVENT_TYPE_MARKER_RECORDED
			if attrs := he.GetMarkerRecordedEventAttributes(); attrs != nil {
				eventName = attrs.MarkerName
			}
		case enums.EVENT_TYPE_WORKFLOW_EXECUTION_SIGNALED:
			eventType = pacev1.HistoryEventType_HISTORY_EVENT_TYPE_WORKFLOW_EXECUTION_SIGNALED
			if attrs := he.GetWorkflowExecutionSignaledEventAttributes(); attrs != nil {
				eventName = "Signal: " + attrs.SignalName
			}
		case enums.EVENT_TYPE_SIGNAL_EXTERNAL_WORKFLOW_EXECUTION_INITIATED:
			eventType = pacev1.HistoryEventType_HISTORY_EVENT_TYPE_SIGNAL_EXTERNAL_WORKFLOW_EXECUTION_INITIATED
		}

		// Don't create empty attributes - frontend checks for null

		events = append(events, &pacev1.HistoryEvent{
			EventId:    he.EventId,
			EventTime:  he.EventTime,
			EventType:  eventType,
			TaskId:     he.TaskId,
			EventName:  eventName,
			Attributes: eventAttrs,
		})
	}

	nextToken := string(resp.NextPageToken)
	return events, nextToken, nil
}

func (s *Service) TerminateWorkflow(ctx context.Context, namespace string, workflowID string, runID string, reason string) error {
	req := &workflowservice.TerminateWorkflowExecutionRequest{
		Namespace: namespace,
		WorkflowExecution: &common.WorkflowExecution{
			WorkflowId: workflowID,
			RunId:      runID,
		},
		Reason: reason,
	}

	_, err := s.client.WorkflowService().TerminateWorkflowExecution(ctx, req)
	if err != nil {
		return fmt.Errorf("terminate workflow: %w", err)
	}

	return nil
}

func (s *Service) CancelWorkflow(ctx context.Context, namespace string, workflowID string, runID string) error {
	req := &workflowservice.RequestCancelWorkflowExecutionRequest{
		Namespace: namespace,
		WorkflowExecution: &common.WorkflowExecution{
			WorkflowId: workflowID,
			RunId:      runID,
		},
	}

	_, err := s.client.WorkflowService().RequestCancelWorkflowExecution(ctx, req)
	if err != nil {
		return fmt.Errorf("cancel workflow: %w", err)
	}

	return nil
}

func (s *Service) SignalWorkflow(ctx context.Context, namespace string, workflowID string, runID string, signalName string, _ *structpb.Struct) error {
	req := &workflowservice.SignalWorkflowExecutionRequest{
		Namespace: namespace,
		WorkflowExecution: &common.WorkflowExecution{
			WorkflowId: workflowID,
			RunId:      runID,
		},
		SignalName: signalName,
		Input:      nil, // Convert structpb.Struct to payloads if needed
	}

	_, err := s.client.WorkflowService().SignalWorkflowExecution(ctx, req)
	if err != nil {
		return fmt.Errorf("signal workflow: %w", err)
	}

	return nil
}

func (s *Service) RestartWorkflow(ctx context.Context, namespace string, workflowID string, runID string) (string, string, error) {
	// First, get the original workflow details to extract input
	descReq := &workflowservice.DescribeWorkflowExecutionRequest{
		Namespace: namespace,
		Execution: &common.WorkflowExecution{
			WorkflowId: workflowID,
			RunId:      runID,
		},
	}

	descResp, err := s.client.WorkflowService().DescribeWorkflowExecution(ctx, descReq)
	if err != nil {
		return "", "", fmt.Errorf("describe workflow for restart: %w", err)
	}

	// Get the workflow history to extract the input from the WorkflowExecutionStarted event
	histReq := &workflowservice.GetWorkflowExecutionHistoryRequest{
		Namespace: namespace,
		Execution: &common.WorkflowExecution{
			WorkflowId: workflowID,
			RunId:      runID,
		},
		MaximumPageSize: 1, // We only need the first event
	}

	histResp, err := s.client.WorkflowService().GetWorkflowExecutionHistory(ctx, histReq)
	if err != nil {
		return "", "", fmt.Errorf("get workflow history for restart: %w", err)
	}

	// Extract input from the first event (WorkflowExecutionStarted)
	var input *common.Payloads
	if len(histResp.History.Events) > 0 {
		startedEvent := histResp.History.Events[0]
		if attrs := startedEvent.GetWorkflowExecutionStartedEventAttributes(); attrs != nil {
			input = attrs.Input
		}
	}

	// Extract task queue from workflow execution info
	taskQueue := "default"
	if descResp.WorkflowExecutionInfo.TaskQueue != "" {
		taskQueue = descResp.WorkflowExecutionInfo.TaskQueue
	}

	// Start a new workflow with the same type and input
	startReq := &workflowservice.StartWorkflowExecutionRequest{
		Namespace:    namespace,
		WorkflowId:   workflowID + "-restart",
		WorkflowType: descResp.WorkflowExecutionInfo.Type,
		Input:        input,
		TaskQueue: &taskqueue.TaskQueue{
			Name: taskQueue,
		},
	}

	startResp, err := s.client.WorkflowService().StartWorkflowExecution(ctx, startReq)
	if err != nil {
		return "", "", fmt.Errorf("start workflow for restart: %w", err)
	}

	return workflowID + "-restart", startResp.RunId, nil
}

func (s *Service) ListSchedules(ctx context.Context, namespace string, pageSize int32, pageToken string) ([]*pacev1.Schedule, string, error) {
	if pageSize == 0 {
		pageSize = 100
	}

	req := &workflowservice.ListSchedulesRequest{
		Namespace:       namespace,
		MaximumPageSize: pageSize,
		NextPageToken:   []byte(pageToken),
	}

	resp, err := s.client.WorkflowService().ListSchedules(ctx, req)
	if err != nil {
		return nil, "", fmt.Errorf("list schedules: %w", err)
	}

	var schedules []*pacev1.Schedule
	for _, sched := range resp.Schedules {
		var cronExpr string
		var nextRunTime, lastRunTime *timestamppb.Timestamp

		// Extract cron expression if available
		if sched.Info != nil && sched.Info.Spec != nil {
			if len(sched.Info.Spec.CronString) > 0 {
				cronExpr = sched.Info.Spec.CronString[0]
			}

			// Get recent actions for last run time
			if sched.Info.RecentActions != nil && len(sched.Info.RecentActions) > 0 {
				lastAction := sched.Info.RecentActions[len(sched.Info.RecentActions)-1]
				if lastAction.ActualTime != nil {
					lastRunTime = lastAction.ActualTime
				}
			}

			// Get future action times for next run
			if sched.Info.FutureActionTimes != nil && len(sched.Info.FutureActionTimes) > 0 {
				nextRunTime = sched.Info.FutureActionTimes[0]
			}
		}

		//paused := false
		notes := ""
		workflowType := ""

		schedules = append(schedules, &pacev1.Schedule{
			ScheduleId:     sched.ScheduleId,
			WorkflowType:   workflowType,
			CronExpression: cronExpr,
			NextRunTime:    nextRunTime,
			LastRunTime:    lastRunTime,
			Paused:         false,
			Notes:          notes,
		})
	}

	nextToken := string(resp.NextPageToken)
	return schedules, nextToken, nil
}

func (s *Service) GetSchedule(ctx context.Context, namespace string, scheduleID string) (*pacev1.Schedule, error) {
	req := &workflowservice.DescribeScheduleRequest{
		Namespace:  namespace,
		ScheduleId: scheduleID,
	}

	resp, err := s.client.WorkflowService().DescribeSchedule(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("describe schedule: %w", err)
	}

	var cronExpr string
	var nextRunTime, lastRunTime *timestamppb.Timestamp

	// Extract cron expression if available
	if resp.Schedule != nil && resp.Schedule.Spec != nil {
		if len(resp.Schedule.Spec.CronString) > 0 {
			cronExpr = resp.Schedule.Spec.CronString[0]
		}
	}

	// Get recent actions for last run time
	if resp.Info != nil {
		if resp.Info.RecentActions != nil && len(resp.Info.RecentActions) > 0 {
			lastAction := resp.Info.RecentActions[len(resp.Info.RecentActions)-1]
			if lastAction.ActualTime != nil {
				lastRunTime = lastAction.ActualTime
			}
		}

		// Get future action times for next run
		if resp.Info.FutureActionTimes != nil && len(resp.Info.FutureActionTimes) > 0 {
			nextRunTime = resp.Info.FutureActionTimes[0]
		}
	}

	//paused := false
	notes := ""
	workflowType := ""

	return &pacev1.Schedule{
		ScheduleId:     scheduleID,
		WorkflowType:   workflowType,
		CronExpression: cronExpr,
		NextRunTime:    nextRunTime,
		LastRunTime:    lastRunTime,
		Paused:         false,
		Notes:          notes,
	}, nil
}

func (s *Service) PauseSchedule(ctx context.Context, namespace string, scheduleID string, note string) error {
	req := &workflowservice.PatchScheduleRequest{
		Namespace:  namespace,
		ScheduleId: scheduleID,
		Patch: &schedule.SchedulePatch{
			Pause: note,
		},
	}

	_, err := s.client.WorkflowService().PatchSchedule(ctx, req)
	if err != nil {
		return fmt.Errorf("pause schedule: %w", err)
	}

	return nil
}

func (s *Service) UnpauseSchedule(ctx context.Context, namespace string, scheduleID string, note string) error {
	req := &workflowservice.PatchScheduleRequest{
		Namespace:  namespace,
		ScheduleId: scheduleID,
		Patch: &schedule.SchedulePatch{
			Unpause: note,
		},
	}

	_, err := s.client.WorkflowService().PatchSchedule(ctx, req)
	if err != nil {
		return fmt.Errorf("unpause schedule: %w", err)
	}

	return nil
}

func (s *Service) TriggerSchedule(ctx context.Context, namespace string, scheduleID string) (string, string, error) {
	req := &workflowservice.PatchScheduleRequest{
		Namespace:  namespace,
		ScheduleId: scheduleID,
		Patch: &schedule.SchedulePatch{
			TriggerImmediately: &schedule.TriggerImmediatelyRequest{},
		},
	}

	_, err := s.client.WorkflowService().PatchSchedule(ctx, req)
	if err != nil {
		return "", "", fmt.Errorf("trigger schedule: %w", err)
	}

	// Return schedule ID as execution ID (actual execution ID will be generated by Temporal)
	return scheduleID + "-triggered", "running", nil
}
