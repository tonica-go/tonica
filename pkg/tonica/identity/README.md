# Identity Package

The identity package provides automatic identity context propagation through Temporal workflows and activities, along with convenient middleware for extracting identity from HTTP requests.

## Features

✅ **Automatic Propagation** - Identity flows through all workflows and activities
✅ **Type-Safe API** - Strongly-typed Identity struct with helper methods
✅ **Flexible Middleware** - Multiple extractors for different auth mechanisms
✅ **Zero Configuration** - Works out of the box with `tonica.GetTemporalClient()`
✅ **Backward Compatible** - Falls back to payload-based identity

## Quick Start

### 1. Add Identity Middleware to Your App

```go
import (
    "github.com/tonica-go/tonica/pkg/tonica"
    "github.com/tonica-go/tonica/pkg/tonica/identity"
)

func main() {
    app := tonica.NewApp()

    // Option 1: Use default extractor (if identity already in gin.Context)
    app.GetRouter().Use(identity.Middleware(identity.DefaultExtractor))

    // Option 2: Extract from JWT claims
    app.GetRouter().Use(identity.Middleware(
        identity.JWTExtractor("claims", "user_id", "email", "role"),
    ))

    // Option 3: Extract from headers
    app.GetRouter().Use(identity.Middleware(
        identity.HeaderExtractor("X-User-ID", "X-User-Email", "X-User-Role"),
    ))

    // Option 4: Chain multiple extractors
    app.GetRouter().Use(identity.Middleware(
        identity.ChainExtractors(
            identity.JWTExtractor("claims", "user_id", "email", "role"),
            identity.HeaderExtractor("X-User-ID", "X-User-Email", "X-User-Role"),
        ),
    ))
}
```

### 2. Trigger Workflows with Identity

```go
func HandleWorkflowTrigger(c *gin.Context) {
    // Identity is automatically in request context!
    ctx := c.Request.Context()

    // Start workflow - identity automatically propagates
    run, err := temporalClient.ExecuteWorkflow(ctx, options, workflowName, input)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, gin.H{"workflow_id": run.GetID()})
}
```

### 3. Access Identity in Workflows

```go
import "github.com/tonica-go/tonica/pkg/tonica/identity"

func MyWorkflow(ctx workflow.Context, input Input) (*Output, error) {
    logger := workflow.GetLogger(ctx)

    // Get identity from context
    id := identity.FromWorkflowContext(ctx)
    if id != nil {
        logger.Info("Workflow started",
            "user_id", id.GetID(),
            "email", id.GetEmail(),
            "role", id.GetRole(),
        )
    }

    // Identity automatically flows to all activities
    var result string
    err := workflow.ExecuteActivity(ctx, MyActivity, input).Get(ctx, &result)

    return &Output{Result: result}, err
}
```

### 4. Access Identity in Activities

```go
import "github.com/tonica-go/tonica/pkg/tonica/identity"

func MyActivity(ctx context.Context, input Input) (string, error) {
    logger := activity.GetLogger(ctx)

    // Identity is automatically available
    id := identity.FromContext(ctx)
    if id != nil {
        logger.Info("Activity executing",
            "user_id", id.GetID(),
            "email", id.GetEmail(),
        )

        // Use for audit logging, permissions, etc.
    }

    return "done", nil
}
```

## Identity API

### Creating Identity

```go
// Minimal identity
id := identity.NewIdentity("user-123")

// Full identity with fluent API
id := identity.NewIdentity("user-123").
    WithEmail("user@example.com").
    WithRole("admin").
    WithName("John Doe").
    WithField("department", "engineering").
    WithField("permissions", []string{"read", "write"})
```

### Accessing Identity Fields

```go
id := identity.FromContext(ctx)

// Built-in getters
userID := id.GetID()      // "user-123"
email := id.GetEmail()    // "user@example.com"
role := id.GetRole()      // "admin"
name := id.GetName()      // "John Doe"

// Access custom fields
dept := id["department"]  // "engineering"
```

### Context Operations

```go
// Add identity to Go context
ctx := identity.ToContext(context.Background(), id)

// Extract from Go context
id := identity.FromContext(ctx)

// Add identity to workflow context
wfCtx := identity.ToWorkflowContext(workflowCtx, id)

// Extract from workflow context
id := identity.FromWorkflowContext(wfCtx)

// Must variants (panic if not found)
id := identity.MustFromContext(ctx)
id := identity.MustFromWorkflowContext(wfCtx)
```

## Middleware Extractors

### Default Extractor

Use when identity is already in gin.Context (e.g., from your auth middleware):

```go
identity.Middleware(identity.DefaultExtractor)
```

### JWT Extractor

Extract identity from JWT claims:

```go
// JWTExtractor(claimsKey, userIDField, emailField, roleField)
identity.Middleware(
    identity.JWTExtractor("claims", "sub", "email", "role"),
)
```

Your JWT middleware should set claims in gin.Context:
```go
func JWTMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Parse JWT and extract claims
        claims := parseJWT(c)
        c.Set("claims", claims) // Set claims in gin.Context
        c.Next()
    }
}
```

### Header Extractor

Extract identity from request headers:

```go
// HeaderExtractor(userIDHeader, emailHeader, roleHeader)
identity.Middleware(
    identity.HeaderExtractor("X-User-ID", "X-User-Email", "X-User-Role"),
)
```

### Chain Extractor

Try multiple extractors in order:

```go
identity.Middleware(
    identity.ChainExtractors(
        identity.JWTExtractor("claims", "sub", "email", "role"),
        identity.HeaderExtractor("X-User-ID", "X-User-Email", "X-User-Role"),
    ),
)
```

### Custom Extractor

Create your own extractor:

```go
func customExtractor(c *gin.Context) identity.Identity {
    // Your custom logic
    userID := extractUserIDSomehow(c)
    if userID == "" {
        return nil
    }

    return identity.NewIdentity(userID).
        WithEmail(extractEmail(c)).
        WithRole(extractRole(c))
}

app.GetRouter().Use(identity.Middleware(customExtractor))
```

## Context Propagator

The `IdentityContextPropagator` implements Temporal's `workflow.ContextPropagator` interface and is automatically registered by `tonica.GetTemporalClient()`.

### How It Works

1. **Client Side (Starting Workflow)**
   - `Inject()` extracts identity from Go context
   - Serializes it to Temporal headers
   - Headers are sent with workflow execution request

2. **Workflow Side (Receiving)**
   - `ExtractToWorkflow()` deserializes from headers
   - Adds identity to workflow context
   - Available via `identity.FromWorkflowContext()`

3. **Activity Side (Receiving)**
   - `Extract()` deserializes from headers
   - Adds identity to activity context
   - Available via `identity.FromContext()`

4. **Child Workflow/Activity (Starting from Workflow)**
   - `InjectFromWorkflow()` extracts from workflow context
   - Identity automatically flows to child executions

## Use Cases

### Audit Logging

```go
func MyActivity(ctx context.Context, input Input) error {
    id := identity.FromContext(ctx)
    if id != nil {
        auditLog.Record(AuditEvent{
            Action:   "record_updated",
            UserID:   id.GetID(),
            UserRole: id.GetRole(),
            Timestamp: time.Now(),
        })
    }
    return nil
}
```

### Permission Checks

```go
func MyActivity(ctx context.Context, input Input) error {
    id := identity.FromContext(ctx)
    if id == nil {
        return fmt.Errorf("unauthorized: no identity")
    }

    if id.GetRole() != "admin" {
        return fmt.Errorf("unauthorized: admin role required")
    }

    // Proceed with privileged operation
    return performAdminAction()
}
```

### Multi-Tenant Operations

```go
func MyActivity(ctx context.Context, input Input) error {
    id := identity.FromContext(ctx)
    tenantID := id["tenant_id"].(string)

    // Use tenant ID for data isolation
    records := db.Query("SELECT * FROM records WHERE tenant_id = ?", tenantID)
    return nil
}
```

## Best Practices

1. **Always Use Middleware** - Add identity middleware early in your router chain
2. **Check for nil** - Always check if identity exists before accessing
3. **Use Type Assertions Safely** - When accessing custom fields, use type assertions with ok checks
4. **Don't Store Sensitive Data** - Only store IDs and metadata, not passwords or tokens
5. **Use Built-in Getters** - Prefer `id.GetID()` over `id["id"]` for better type safety

## Troubleshooting

### Identity is nil in workflow/activity

**Cause**: Identity not set when starting workflow

**Solution**: Ensure middleware is registered and context is passed:
```go
// Middleware must be registered
app.GetRouter().Use(identity.Middleware(identity.DefaultExtractor))

// Use request context when starting workflow
ctx := c.Request.Context() // from gin.Context
run, err := temporalClient.ExecuteWorkflow(ctx, options, workflowName, input)
```

### Identity not propagating to child workflows

**Cause**: Context propagator not registered (this should never happen with tonica)

**Solution**: Ensure you're using `tonica.GetTemporalClient()` or `tonica.MustGetTemporalClient()`

### Custom fields not available

**Cause**: Extractor not adding custom fields

**Solution**: Use `WithField()` when creating identity:
```go
id := identity.NewIdentity(userID).
    WithField("custom_field", value)
```

## Example: Complete Integration

```go
package main

import (
    "github.com/tonica-go/tonica/pkg/tonica"
    "github.com/tonica-go/tonica/pkg/tonica/identity"
    "go.temporal.io/sdk/client"
)

func main() {
    // Create app
    app := tonica.NewApp()

    // Add identity middleware (extracts from JWT)
    app.GetRouter().Use(identity.Middleware(
        identity.JWTExtractor("claims", "sub", "email", "role"),
    ))

    // Register routes
    app.GetRouter().POST("/trigger-workflow", handleTriggerWorkflow)

    // Start app
    app.Start()
}

func handleTriggerWorkflow(c *gin.Context) {
    // Get temporal client (has identity propagator registered)
    temporalClient := tonica.MustGetTemporalClient("")

    // Use request context (contains identity from middleware)
    ctx := c.Request.Context()

    // Start workflow - identity automatically propagates!
    options := client.StartWorkflowOptions{
        ID:        "workflow-123",
        TaskQueue: "default",
    }

    run, err := temporalClient.ExecuteWorkflow(ctx, options, MyWorkflow, input)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, gin.H{"workflow_id": run.GetID()})
}

func MyWorkflow(ctx workflow.Context, input Input) (*Output, error) {
    // Identity is automatically available!
    id := identity.FromWorkflowContext(ctx)

    logger := workflow.GetLogger(ctx)
    logger.Info("Workflow executing", "user_id", id.GetID())

    // Execute activities - identity flows automatically
    var result string
    err := workflow.ExecuteActivity(ctx, MyActivity, input).Get(ctx, &result)

    return &Output{Result: result}, err
}

func MyActivity(ctx context.Context, input Input) (string, error) {
    // Identity is automatically available!
    id := identity.FromContext(ctx)

    logger := activity.GetLogger(ctx)
    logger.Info("Activity executing",
        "user_id", id.GetID(),
        "email", id.GetEmail(),
    )

    // Use identity for audit, permissions, etc.
    return "done", nil
}
```

## Migration from Old Code

### Before (manual identity in payload)

```go
// In handler
input := WorkflowInput{
    Payload: map[string]any{
        "triggeredBy": userID,
        "data": data,
    },
}

// In activity
func MyActivity(ctx context.Context, input WorkflowInput) error {
    userID := input.Payload["triggeredBy"].(string)
    // use userID
}
```

### After (automatic identity propagation)

```go
// In handler
ctx := c.Request.Context() // identity from middleware
input := WorkflowInput{
    Payload: map[string]any{
        "data": data,
    },
}

// In activity
func MyActivity(ctx context.Context, input WorkflowInput) error {
    id := identity.FromContext(ctx)
    userID := id.GetID()
    // use userID
}
```

## Advanced Topics

### Custom Context Propagator

If you need additional context propagation beyond identity:

```go
type CustomPropagator struct{}

func (p *CustomPropagator) Inject(ctx context.Context, writer workflow.HeaderWriter) error {
    // Extract custom data from context
    // Write to headers
}

// Implement other methods...

// Register with temporal client
opts.ContextPropagators = []workflow.ContextPropagator{
    identity.NewIdentityContextPropagator(),
    &CustomPropagator{},
}
```

### Testing with Identity

```go
func TestWorkflowWithIdentity(t *testing.T) {
    // Create test identity
    testID := identity.NewIdentity("test-user").
        WithEmail("test@example.com").
        WithRole("admin")

    // Add to context
    ctx := identity.ToContext(context.Background(), testID)

    // Start workflow with identity
    env.ExecuteWorkflow(MyWorkflow, input)

    // Identity will be available in workflow and activities
}
```

## See Also

- [Temporal Context Propagation](https://docs.temporal.io/develop/go/observability)
- [Gin Middleware](https://gin-gonic.com/docs/examples/custom-middleware/)
- [Context Package](https://pkg.go.dev/context)
