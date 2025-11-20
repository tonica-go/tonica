package tonica

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRoute(t *testing.T) {
	app := NewApp()
	rb := NewRoute(app)

	assert.NotNil(t, rb)
	assert.Same(t, app, rb.app)
	assert.NotNil(t, rb.responses)
	assert.Empty(t, rb.responses)
}

func TestRouteBuilder_HTTPMethods(t *testing.T) {
	app := NewApp()

	tests := []struct {
		name   string
		method func(*RouteBuilder, string) *RouteBuilder
		want   string
	}{
		{"GET", (*RouteBuilder).GET, "GET"},
		{"POST", (*RouteBuilder).POST, "POST"},
		{"PUT", (*RouteBuilder).PUT, "PUT"},
		{"PATCH", (*RouteBuilder).PATCH, "PATCH"},
		{"DELETE", (*RouteBuilder).DELETE, "DELETE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := NewRoute(app)
			result := tt.method(rb, "/test")

			assert.Equal(t, tt.want, rb.method)
			assert.Equal(t, "/test", rb.path)
			assert.Same(t, rb, result, "should return self for chaining")
		})
	}
}

func TestRouteBuilder_Metadata(t *testing.T) {
	app := NewApp()

	t.Run("Summary", func(t *testing.T) {
		rb := NewRoute(app).Summary("Test summary")
		assert.Equal(t, "Test summary", rb.summary)
	})

	t.Run("Description", func(t *testing.T) {
		rb := NewRoute(app).Description("Test description")
		assert.Equal(t, "Test description", rb.description)
	})

	t.Run("Tag", func(t *testing.T) {
		rb := NewRoute(app).Tag("Tag1").Tag("Tag2")
		assert.Equal(t, []string{"Tag1", "Tag2"}, rb.tags)
	})

	t.Run("Tags", func(t *testing.T) {
		rb := NewRoute(app).Tags("Tag1", "Tag2", "Tag3")
		assert.Equal(t, []string{"Tag1", "Tag2", "Tag3"}, rb.tags)
	})
}

func TestRouteBuilder_Parameters(t *testing.T) {
	app := NewApp()

	t.Run("QueryParam", func(t *testing.T) {
		rb := NewRoute(app).
			QueryParam("name", "string", "User name", true).
			QueryParam("age", "integer", "User age", false)

		require.Len(t, rb.parameters, 2)
		assert.Equal(t, "name", rb.parameters[0].Name)
		assert.Equal(t, "query", rb.parameters[0].In)
		assert.Equal(t, "string", rb.parameters[0].Type)
		assert.True(t, rb.parameters[0].Required)

		assert.Equal(t, "age", rb.parameters[1].Name)
		assert.False(t, rb.parameters[1].Required)
	})

	t.Run("PathParam", func(t *testing.T) {
		rb := NewRoute(app).PathParam("id", "string", "User ID")

		require.Len(t, rb.parameters, 1)
		assert.Equal(t, "id", rb.parameters[0].Name)
		assert.Equal(t, "path", rb.parameters[0].In)
		assert.True(t, rb.parameters[0].Required, "path params should always be required")
	})

	t.Run("HeaderParam", func(t *testing.T) {
		rb := NewRoute(app).HeaderParam("Authorization", "string", "Auth token", true)

		require.Len(t, rb.parameters, 1)
		assert.Equal(t, "Authorization", rb.parameters[0].Name)
		assert.Equal(t, "header", rb.parameters[0].In)
	})

	t.Run("BodyParam", func(t *testing.T) {
		schema := map[string]interface{}{"type": "object"}
		rb := NewRoute(app).BodyParam("User data", schema)

		require.Len(t, rb.parameters, 1)
		assert.Equal(t, "body", rb.parameters[0].In)
		assert.Equal(t, "User data", rb.parameters[0].Description)
		assert.Equal(t, schema, rb.parameters[0].Schema)
		assert.True(t, rb.parameters[0].Required)
	})

	t.Run("FormParam", func(t *testing.T) {
		rb := NewRoute(app).FormParam("username", "Username", "string")

		require.Len(t, rb.parameters, 1)
		assert.Equal(t, "username", rb.parameters[0].Name)
		assert.Equal(t, "formData", rb.parameters[0].In)
		assert.Equal(t, "string", rb.parameters[0].Type)
	})

	t.Run("FormFileParam", func(t *testing.T) {
		rb := NewRoute(app).FormFileParam("avatar", "User avatar file")

		require.Len(t, rb.parameters, 1)
		assert.Equal(t, "avatar", rb.parameters[0].Name)
		assert.Equal(t, "formData", rb.parameters[0].In)
		assert.Equal(t, "file", rb.parameters[0].Type)
	})
}

func TestRouteBuilder_Response(t *testing.T) {
	app := NewApp()

	schema := map[string]string{"type": "object"}
	rb := NewRoute(app).
		Response(200, "Success", schema).
		Response(404, "Not found", nil)

	require.Len(t, rb.responses, 2)
	assert.Equal(t, "Success", rb.responses["200"].Description)
	assert.Equal(t, schema, rb.responses["200"].Schema)
	assert.Equal(t, "Not found", rb.responses["404"].Description)
}

func TestRouteBuilder_Security(t *testing.T) {
	app := NewApp()

	rb := NewRoute(app).
		Security("bearer").
		Security("apiKey", "read", "write")

	require.Len(t, rb.security, 2)
	assert.Equal(t, []string(nil), rb.security[0]["bearer"])
	assert.Equal(t, []string{"read", "write"}, rb.security[1]["apiKey"])
}

func TestRouteBuilder_Handle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := NewApp()

	t.Run("should register route with Gin", func(t *testing.T) {
		called := false
		NewRoute(app).
			GET("/test").
			Handle(func(c *gin.Context) {
				called = true
				c.JSON(200, gin.H{"status": "ok"})
			})

		// Test the route
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		app.router.ServeHTTP(w, req)

		assert.True(t, called)
		assert.Equal(t, 200, w.Code)
		assert.JSONEq(t, `{"status":"ok"}`, w.Body.String())
	})

	t.Run("should store metadata", func(t *testing.T) {
		testApp := NewApp()

		NewRoute(testApp).
			POST("/users").
			Summary("Create user").
			Tags("Users").
			BodyParam("User data", InlineObjectSchema(map[string]string{"name": "string"})).
			Response(201, "Created", nil).
			Handle(func(c *gin.Context) {
				c.Status(201)
			})

		require.Len(t, testApp.customRoutes, 1)
		metadata := testApp.customRoutes[0]

		assert.Equal(t, "POST", metadata.Method)
		assert.Equal(t, "/users", metadata.Path)
		assert.Equal(t, "Create user", metadata.Summary)
		assert.Contains(t, metadata.Tags, "Users")
		assert.NotEmpty(t, metadata.OperationID)
		assert.Len(t, metadata.Parameters, 1)
		assert.Len(t, metadata.Responses, 1)
	})

	t.Run("should panic without method", func(t *testing.T) {
		assert.Panics(t, func() {
			NewRoute(app).Handle(func(c *gin.Context) {})
		})
	})
}

func TestRouteBuilder_FluentAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := NewApp()

	// Test full fluent chain
	NewRoute(app).
		GET("/users/:id").
		Summary("Get user").
		Description("Retrieves a user by ID").
		Tags("Users", "Public").
		PathParam("id", "string", "User ID").
		QueryParam("fields", "string", "Fields to return", false).
		Security("bearer").
		Response(200, "Success", InlineObjectSchema(map[string]string{
			"id":   "string",
			"name": "string",
		})).
		Response(404, "Not found", nil).
		Handle(func(c *gin.Context) {
			c.JSON(200, gin.H{"id": c.Param("id")})
		})

	// Verify route works
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/users/123", nil)
	app.router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	// Verify metadata
	require.Len(t, app.customRoutes, 1)
	metadata := app.customRoutes[0]

	assert.Equal(t, "GET", metadata.Method)
	assert.Equal(t, "/users/:id", metadata.Path)
	assert.Equal(t, "Get user", metadata.Summary)
	assert.Equal(t, "Retrieves a user by ID", metadata.Description)
	assert.Equal(t, []string{"Users", "Public"}, metadata.Tags)
	assert.Len(t, metadata.Parameters, 2)
	assert.Len(t, metadata.Responses, 2)
	assert.Len(t, metadata.Security, 1)
}

func TestSchemaHelpers(t *testing.T) {
	t.Run("StringSchema", func(t *testing.T) {
		schema := StringSchema()
		assert.Equal(t, "string", schema["type"])
	})

	t.Run("ObjectSchema", func(t *testing.T) {
		props := map[string]interface{}{
			"name": map[string]string{"type": "string"},
			"age":  map[string]string{"type": "integer"},
		}
		schema := ObjectSchema(props)

		assert.Equal(t, "object", schema["type"])
		assert.Equal(t, props, schema["properties"])
	})

	t.Run("ArraySchema", func(t *testing.T) {
		items := map[string]string{"type": "string"}
		schema := ArraySchema(items)

		assert.Equal(t, "array", schema["type"])
		assert.Equal(t, items, schema["items"])
	})

	t.Run("RefSchema", func(t *testing.T) {
		schema := RefSchema("User")
		assert.Equal(t, "#/definitions/User", schema["$ref"])
	})

	t.Run("InlineObjectSchema", func(t *testing.T) {
		schema := InlineObjectSchema(map[string]string{
			"name":  "string",
			"email": "string",
			"age":   "integer",
		})

		assert.Equal(t, "object", schema["type"])
		props := schema["properties"].(map[string]interface{})
		assert.Equal(t, "string", props["name"].(map[string]string)["type"])
		assert.Equal(t, "integer", props["age"].(map[string]string)["type"])
	})
}

func TestMergeCustomRoutesIntoSpec(t *testing.T) {
	t.Run("should merge custom routes into empty spec", func(t *testing.T) {
		specBytes := []byte(`{"swagger":"2.0","paths":{}}`)

		routes := []RouteMetadata{
			{
				Method:  "GET",
				Path:    "/test",
				Summary: "Test endpoint",
				Tags:    []string{"Test"},
				Responses: map[string]RouteResponse{
					"200": {Description: "Success"},
				},
			},
		}

		result, err := mergeCustomRoutesIntoSpec(specBytes, routes)
		require.NoError(t, err)

		var spec map[string]interface{}
		err = json.Unmarshal(result, &spec)
		require.NoError(t, err)

		paths := spec["paths"].(map[string]interface{})
		testPath := paths["/test"].(map[string]interface{})
		getOp := testPath["get"].(map[string]interface{})

		assert.Equal(t, "Test endpoint", getOp["summary"])
		assert.Contains(t, getOp["tags"], "Test")
	})

	t.Run("should preserve existing paths", func(t *testing.T) {
		specBytes := []byte(`{
			"swagger":"2.0",
			"paths":{
				"/existing":{
					"get":{"summary":"Existing endpoint"}
				}
			}
		}`)

		routes := []RouteMetadata{
			{
				Method:  "POST",
				Path:    "/new",
				Summary: "New endpoint",
			},
		}

		result, err := mergeCustomRoutesIntoSpec(specBytes, routes)
		require.NoError(t, err)

		var spec map[string]interface{}
		err = json.Unmarshal(result, &spec)
		require.NoError(t, err)

		paths := spec["paths"].(map[string]interface{})
		assert.Contains(t, paths, "/existing")
		assert.Contains(t, paths, "/new")
	})

	t.Run("should handle empty custom routes", func(t *testing.T) {
		specBytes := []byte(`{"swagger":"2.0","paths":{}}`)

		result, err := mergeCustomRoutesIntoSpec(specBytes, []RouteMetadata{})
		require.NoError(t, err)
		assert.Equal(t, specBytes, result)
	})

	t.Run("should handle multiple methods on same path", func(t *testing.T) {
		specBytes := []byte(`{"swagger":"2.0","paths":{}}`)

		routes := []RouteMetadata{
			{
				Method:  "GET",
				Path:    "/users",
				Summary: "List users",
			},
			{
				Method:  "POST",
				Path:    "/users",
				Summary: "Create user",
			},
		}

		result, err := mergeCustomRoutesIntoSpec(specBytes, routes)
		require.NoError(t, err)

		var spec map[string]interface{}
		err = json.Unmarshal(result, &spec)
		require.NoError(t, err)

		paths := spec["paths"].(map[string]interface{})
		usersPath := paths["/users"].(map[string]interface{})

		assert.Contains(t, usersPath, "get")
		assert.Contains(t, usersPath, "post")
	})
}

func TestRouteBuilder_generateOperationID(t *testing.T) {
	app := NewApp()

	tests := []struct {
		name     string
		method   string
		path     string
		expected string
	}{
		{"simple GET", "GET", "/users", "GETusers"},
		{"with path param", "GET", "/users/:id", "GETusersid"},
		{"with multiple segments", "POST", "/api/v1/users", "POSTapiv1users"},
		{"with special chars", "PUT", "/users-list", "PUTuserslist"},
		{"with underscores", "DELETE", "/user_profiles", "DELETEuserprofiles"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := &RouteBuilder{
				app:    app,
				method: tt.method,
				path:   tt.path,
			}

			result := rb.generateOperationID()
			assert.Equal(t, tt.expected, result)
		})
	}
}
