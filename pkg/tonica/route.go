package tonica

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
)

// RouteBuilder provides fluent API for creating documented custom routes
type RouteBuilder struct {
	app         *App
	method      string
	path        string
	summary     string
	description string
	tags        []string
	parameters  []RouteParameter
	responses   map[string]RouteResponse
	security    []map[string][]string
	handler     gin.HandlerFunc
}

// RouteParameter represents an OpenAPI parameter
type RouteParameter struct {
	Name        string      `json:"name"`
	In          string      `json:"in"` // query, path, header, body
	Description string      `json:"description,omitempty"`
	Required    bool        `json:"required,omitempty"`
	Type        string      `json:"type,omitempty"`   // string, number, integer, boolean, array, object
	Schema      interface{} `json:"schema,omitempty"` // for body parameters
	Format      string      `json:"format,omitempty"` // int32, int64, float, double, etc.
	Default     interface{} `json:"default,omitempty"`
}

// RouteResponse represents an OpenAPI response
type RouteResponse struct {
	Description string      `json:"description"`
	Schema      interface{} `json:"schema,omitempty"`
}

// RouteMetadata contains all OpenAPI metadata for a custom route
type RouteMetadata struct {
	Method      string                   `json:"-"`
	Path        string                   `json:"-"`
	Summary     string                   `json:"summary,omitempty"`
	Description string                   `json:"description,omitempty"`
	Tags        []string                 `json:"tags,omitempty"`
	OperationID string                   `json:"operationId,omitempty"`
	Parameters  []RouteParameter         `json:"parameters,omitempty"`
	Responses   map[string]RouteResponse `json:"responses"`
	Security    []map[string][]string    `json:"security,omitempty"`
}

// NewRoute creates a new route builder
func NewRoute(app *App) *RouteBuilder {
	return &RouteBuilder{
		app:       app,
		responses: make(map[string]RouteResponse),
	}
}

// GET sets the HTTP method to GET
func (rb *RouteBuilder) GET(path string) *RouteBuilder {
	rb.method = "GET"
	rb.path = path
	return rb
}

// POST sets the HTTP method to POST
func (rb *RouteBuilder) POST(path string) *RouteBuilder {
	rb.method = "POST"
	rb.path = path
	return rb
}

// PUT sets the HTTP method to PUT
func (rb *RouteBuilder) PUT(path string) *RouteBuilder {
	rb.method = "PUT"
	rb.path = path
	return rb
}

// PATCH sets the HTTP method to PATCH
func (rb *RouteBuilder) PATCH(path string) *RouteBuilder {
	rb.method = "PATCH"
	rb.path = path
	return rb
}

// DELETE sets the HTTP method to DELETE
func (rb *RouteBuilder) DELETE(path string) *RouteBuilder {
	rb.method = "DELETE"
	rb.path = path
	return rb
}

// Summary sets the route summary
func (rb *RouteBuilder) Summary(summary string) *RouteBuilder {
	rb.summary = summary
	return rb
}

// Description sets the route description
func (rb *RouteBuilder) Description(description string) *RouteBuilder {
	rb.description = description
	return rb
}

// Tag adds a tag to the route
func (rb *RouteBuilder) Tag(tag string) *RouteBuilder {
	rb.tags = append(rb.tags, tag)
	return rb
}

// Tags sets multiple tags at once
func (rb *RouteBuilder) Tags(tags ...string) *RouteBuilder {
	rb.tags = tags
	return rb
}

// QueryParam adds a query parameter
func (rb *RouteBuilder) QueryParam(name, paramType, description string, required bool) *RouteBuilder {
	rb.parameters = append(rb.parameters, RouteParameter{
		Name:        name,
		In:          "query",
		Type:        paramType,
		Description: description,
		Required:    required,
	})
	return rb
}

// PathParam adds a path parameter
func (rb *RouteBuilder) PathParam(name, paramType, description string) *RouteBuilder {
	rb.parameters = append(rb.parameters, RouteParameter{
		Name:        name,
		In:          "path",
		Type:        paramType,
		Description: description,
		Required:    true,
	})
	return rb
}

// HeaderParam adds a header parameter
func (rb *RouteBuilder) HeaderParam(name, paramType, description string, required bool) *RouteBuilder {
	rb.parameters = append(rb.parameters, RouteParameter{
		Name:        name,
		In:          "header",
		Type:        paramType,
		Description: description,
		Required:    required,
	})
	return rb
}

// BodyParam adds a body parameter with schema
func (rb *RouteBuilder) BodyParam(description string, schema interface{}) *RouteBuilder {
	rb.parameters = append(rb.parameters, RouteParameter{
		Name:        "body",
		In:          "body",
		Description: description,
		Required:    true,
		Schema:      schema,
	})
	return rb
}

// FormParam adds a form data parameter
func (rb *RouteBuilder) FormParam(name, description, typ string) *RouteBuilder {
	rb.parameters = append(rb.parameters, RouteParameter{
		Name:        name,
		In:          "formData",
		Description: description,
		Required:    true,
		Type:        typ,
	})
	return rb
}

// FormFileParam adds a form data file parameter
func (rb *RouteBuilder) FormFileParam(name, description string) *RouteBuilder {
	rb.parameters = append(rb.parameters, RouteParameter{
		Name:        name,
		In:          "formData",
		Description: description,
		Required:    true,
		Type:        "file",
	})
	return rb
}

// Response adds a response definition
func (rb *RouteBuilder) Response(statusCode int, description string, schema interface{}) *RouteBuilder {
	rb.responses[fmt.Sprintf("%d", statusCode)] = RouteResponse{
		Description: description,
		Schema:      schema,
	}
	return rb
}

// Security adds security requirement (e.g., bearer token)
func (rb *RouteBuilder) Security(name string, scopes ...string) *RouteBuilder {
	securityReq := map[string][]string{
		name: scopes,
	}
	rb.security = append(rb.security, securityReq)
	return rb
}

// Handle registers the handler and metadata
func (rb *RouteBuilder) Handle(handler gin.HandlerFunc) {
	if rb.method == "" || rb.path == "" {
		panic("route method and path must be set before calling Handle")
	}

	rb.handler = handler

	// Register the route with Gin
	switch rb.method {
	case "GET":
		rb.app.router.GET(rb.path, handler)
	case "POST":
		rb.app.router.POST(rb.path, handler)
	case "PUT":
		rb.app.router.PUT(rb.path, handler)
	case "PATCH":
		rb.app.router.PATCH(rb.path, handler)
	case "DELETE":
		rb.app.router.DELETE(rb.path, handler)
	default:
		panic(fmt.Sprintf("unsupported HTTP method: %s", rb.method))
	}

	// Store metadata for OpenAPI spec generation
	metadata := RouteMetadata{
		Method:      rb.method,
		Path:        rb.path,
		Summary:     rb.summary,
		Description: rb.description,
		Tags:        rb.tags,
		OperationID: rb.generateOperationID(),
		Parameters:  rb.parameters,
		Responses:   rb.responses,
		Security:    rb.security,
	}

	rb.app.customRoutes = append(rb.app.customRoutes, metadata)
}

// generateOperationID creates an operation ID from method and path
func (rb *RouteBuilder) generateOperationID() string {
	// Simple operation ID generation: Method + sanitized path
	// e.g., GET /test -> GetTest
	operationID := rb.method
	for _, char := range rb.path {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') {
			operationID += string(char)
		} else if char == '/' || char == '-' || char == '_' {
			// Skip separators for now
			continue
		}
	}
	return operationID
}

// Helper functions to create common schema types

// StringSchema creates a simple string schema
func StringSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "string",
	}
}

// ObjectSchema creates an object schema with properties
func ObjectSchema(properties map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
}

// ArraySchema creates an array schema
func ArraySchema(items interface{}) map[string]interface{} {
	return map[string]interface{}{
		"type":  "array",
		"items": items,
	}
}

// RefSchema creates a reference to a definition
func RefSchema(ref string) map[string]interface{} {
	return map[string]interface{}{
		"$ref": fmt.Sprintf("#/definitions/%s", ref),
	}
}

// InlineObjectSchema creates inline object with simple properties
func InlineObjectSchema(props map[string]string) map[string]interface{} {
	properties := make(map[string]interface{})
	for key, valueType := range props {
		properties[key] = map[string]string{"type": valueType}
	}
	return ObjectSchema(properties)
}

// mergeCustomRoutesIntoSpec merges custom route metadata into the OpenAPI spec
func mergeCustomRoutesIntoSpec(specBytes []byte, customRoutes []RouteMetadata) ([]byte, error) {
	if len(customRoutes) == 0 {
		return specBytes, nil
	}

	var spec map[string]interface{}
	if err := json.Unmarshal(specBytes, &spec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal spec: %w", err)
	}

	// Ensure paths exists
	paths, ok := spec["paths"].(map[string]interface{})
	if !ok {
		paths = make(map[string]interface{})
		spec["paths"] = paths
	}

	// Add custom routes
	for _, route := range customRoutes {
		// Get or create path object
		pathObj, ok := paths[route.Path].(map[string]interface{})
		if !ok {
			pathObj = make(map[string]interface{})
			paths[route.Path] = pathObj
		}

		// Create operation object
		operation := make(map[string]interface{})
		if route.Summary != "" {
			operation["summary"] = route.Summary
		}
		if route.Description != "" {
			operation["description"] = route.Description
		}
		if len(route.Tags) > 0 {
			operation["tags"] = route.Tags
		}
		if route.OperationID != "" {
			operation["operationId"] = route.OperationID
		}
		if len(route.Parameters) > 0 {
			operation["parameters"] = route.Parameters
		}
		if len(route.Responses) > 0 {
			operation["responses"] = route.Responses
		}
		if len(route.Security) > 0 {
			operation["security"] = route.Security
		}

		// Add operation to path with lowercase method
		methodKey := ""
		switch route.Method {
		case "GET":
			methodKey = "get"
		case "POST":
			methodKey = "post"
		case "PUT":
			methodKey = "put"
		case "PATCH":
			methodKey = "patch"
		case "DELETE":
			methodKey = "delete"
		}
		pathObj[methodKey] = operation
	}

	// Marshal back to JSON
	return json.MarshalIndent(spec, "", "  ")
}
