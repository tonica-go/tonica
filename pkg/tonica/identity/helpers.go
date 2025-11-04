package identity

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

// Identity represents user identity information
type Identity map[string]interface{}

// GetID returns the user ID from identity
func (i Identity) GetID() string {
	if id, ok := i["id"].(string); ok {
		return id
	}
	return ""
}

// GetEmail returns the user email from identity
func (i Identity) GetEmail() string {
	if email, ok := i["email"].(string); ok {
		return email
	}
	return ""
}

// GetRole returns the user role from identity
func (i Identity) GetRole() string {
	if role, ok := i["role"].(string); ok {
		return role
	}
	return ""
}

// GetName returns the user name from identity
func (i Identity) GetName() string {
	if name, ok := i["name"].(string); ok {
		return name
	}
	return ""
}

// FromContext extracts identity from Go context
// Returns nil if identity is not found
func FromContext(ctx context.Context) Identity {
	identity := ctx.Value(IdentityContextKey)
	if identity == nil {
		return nil
	}

	// Try to convert to Identity map
	if identityMap, ok := identity.(map[string]interface{}); ok {
		return Identity(identityMap)
	}

	return nil
}

// FromWorkflowContext extracts identity from Temporal workflow context
// Returns nil if identity is not found
func FromWorkflowContext(ctx workflow.Context) Identity {
	identity := ctx.Value(IdentityContextKey)
	if identity == nil {
		return nil
	}

	// Try to convert to Identity map
	if identityMap, ok := identity.(map[string]interface{}); ok {
		return Identity(identityMap)
	}

	return nil
}

// ToContext adds identity to Go context
func ToContext(ctx context.Context, identity Identity) context.Context {
	if identity == nil {
		return ctx
	}
	return context.WithValue(ctx, IdentityContextKey, map[string]interface{}(identity))
}

// ToWorkflowContext adds identity to Temporal workflow context
func ToWorkflowContext(ctx workflow.Context, identity Identity) workflow.Context {
	if identity == nil {
		return ctx
	}
	return workflow.WithValue(ctx, IdentityContextKey, map[string]interface{}(identity))
}

// NewIdentity creates a new Identity with minimal required information
func NewIdentity(userID string) Identity {
	return Identity{
		"id": userID,
	}
}

// WithEmail adds email to identity
func (i Identity) WithEmail(email string) Identity {
	i["email"] = email
	return i
}

// WithRole adds role to identity
func (i Identity) WithRole(role string) Identity {
	i["role"] = role
	return i
}

// WithName adds name to identity
func (i Identity) WithName(name string) Identity {
	i["name"] = name
	return i
}

// WithField adds custom field to identity
func (i Identity) WithField(key string, value interface{}) Identity {
	i[key] = value
	return i
}

// MustFromContext extracts identity from context and panics if not found
// Use this only when identity is guaranteed to exist
func MustFromContext(ctx context.Context) Identity {
	identity := FromContext(ctx)
	if identity == nil {
		panic("identity not found in context")
	}
	return identity
}

// MustFromWorkflowContext extracts identity from workflow context and panics if not found
// Use this only when identity is guaranteed to exist
func MustFromWorkflowContext(ctx workflow.Context) Identity {
	identity := FromWorkflowContext(ctx)
	if identity == nil {
		panic("identity not found in workflow context")
	}
	return identity
}
