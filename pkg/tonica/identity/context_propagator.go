package identity

import (
	"context"

	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/workflow"
)

const (
	// IdentityContextKey is the key for identity in context
	IdentityContextKey = "identity"
)

// IdentityContextPropagator propagates identity context through Temporal workflows
// This allows authentication information to flow through workflows and activities
// without polluting the workflow payload
type IdentityContextPropagator struct{}

// NewIdentityContextPropagator creates a new identity context propagator
func NewIdentityContextPropagator() *IdentityContextPropagator {
	return &IdentityContextPropagator{}
}

// Inject extracts identity from Go context and adds it to Temporal headers
// This is called when starting a workflow or activity from regular Go code
func (p *IdentityContextPropagator) Inject(ctx context.Context, writer workflow.HeaderWriter) error {
	identity := ctx.Value(IdentityContextKey)
	if identity == nil {
		return nil
	}

	// Convert identity to payload and write to headers
	payload, err := converter.GetDefaultDataConverter().ToPayload(identity)
	if err != nil {
		return err
	}

	writer.Set(IdentityContextKey, payload)
	return nil
}

// InjectFromWorkflow extracts identity from workflow context and adds it to headers
// This is called when starting a child workflow or activity from within a workflow
func (p *IdentityContextPropagator) InjectFromWorkflow(ctx workflow.Context, writer workflow.HeaderWriter) error {
	identity := ctx.Value(IdentityContextKey)
	if identity == nil {
		return nil
	}

	// Convert identity to payload and write to headers
	payload, err := converter.GetDefaultDataConverter().ToPayload(identity)
	if err != nil {
		return err
	}

	writer.Set(IdentityContextKey, payload)
	return nil
}

// Extract retrieves identity from Temporal headers and adds it to Go context
// This is called when receiving a workflow or activity in regular Go code
func (p *IdentityContextPropagator) Extract(ctx context.Context, reader workflow.HeaderReader) (context.Context, error) {
	payload, ok := reader.Get(IdentityContextKey)
	if !ok {
		return ctx, nil
	}

	var identity interface{}
	if err := converter.GetDefaultDataConverter().FromPayload(payload, &identity); err != nil {
		return ctx, err
	}

	return context.WithValue(ctx, IdentityContextKey, identity), nil
}

// ExtractToWorkflow retrieves identity from headers and adds it to workflow context
// This is called when receiving a child workflow from a parent workflow
func (p *IdentityContextPropagator) ExtractToWorkflow(ctx workflow.Context, reader workflow.HeaderReader) (workflow.Context, error) {
	payload, ok := reader.Get(IdentityContextKey)
	if !ok {
		return ctx, nil
	}

	var identity interface{}
	if err := converter.GetDefaultDataConverter().FromPayload(payload, &identity); err != nil {
		return ctx, err
	}

	return workflow.WithValue(ctx, IdentityContextKey, identity), nil
}
