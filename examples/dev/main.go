package main

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	paymentv1 "github.com/tonica-go/tonica/examples/dev/proto/payment/v1"
	reportsv1 "github.com/tonica-go/tonica/examples/dev/proto/reports/v1"
	"github.com/tonica-go/tonica/examples/dev/services/payment"
	"github.com/tonica-go/tonica/examples/dev/services/report"
	"github.com/tonica-go/tonica/pkg/tonica"
	"github.com/tonica-go/tonica/pkg/tonica/config"
	"github.com/tonica-go/tonica/pkg/tonica/service"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	app := tonica.NewApp(
		tonica.WithSpec("example/dev/openapi/openapi.swagger.json"),
		tonica.WithSpecUrl("http://localhost:8080/openapi.json"),
		tonica.WithConfig(
			config.NewConfig(
				config.WithRunMode(
					config.GetEnv("APP_MODE", config.ModeAIO),
				),
				config.WithServices(
					strings.Split(",", config.GetEnv("APP_SERVICES", "")),
				),
			),
		),
	)
	customRouter(app)
	initServices(app)
	err := app.Run()
	if err != nil {
		app.GetLogger().Fatal(err)
	}
}

func initServices(app *tonica.App) {
	paymentSrc := service.NewService(
		service.WithName(paymentv1.ServiceName),
		service.WithGRPC(payment.RegisterGRPC),
		service.WithGRPCAddr(config.GetEnv(paymentv1.ServiceAddrEnvName, ":9000")),
		service.WithGateway(payment.RegisterGateway),
	)
	app.GetRegistry().MustRegisterService(paymentSrc)

	app.GetRegistry().MustRegisterService(
		service.NewService(
			service.WithName(reportsv1.ServiceName),
			service.WithGRPC(report.RegisterGRPC),
			service.WithGRPCAddr(config.GetEnv(reportsv1.ServiceAddrEnvName, ":9001")),
		),
	)

	//app.GetRegistry().MustRegisterConsumer(
	//	consumer.NewConsumer(
	//		consumer.WithName("payments"),
	//		consumer.WithClient(kafka.New(&kafka.Config{
	//			Brokers:         []string{config.GetEnv("KAFKA_BROKERS", "")},
	//			ConsumerGroupID: config.GetEnv("KAFKA_CONSUMER_GROUP_ID", "payments"),
	//			BatchBytes:      config.GetEnvInt("KAFKA_BATCH_BYTES", 5000),
	//			BatchTimeout:    config.GetEnvInt("KAFKA_BATCH_TIMEOUT", 5000),
	//			BatchSize:       config.GetEnvInt("KAFKA_BATCH_SIZE", 5000),
	//			TLS:             kafka.TLSConfig{},
	//		}, app.GetMetricManager())),
	//		consumer.WithHandler(payment.GetConsumer(paymentSrc)),
	//	),
	//)

	//cs, _ := app.GetRegistry().GetConsumer("payments")
	//_ = cs.GetClient().Publish(ctx, "payments", []byte("Hello World"))
}

//
//func testClient(app *tonica.App) {
//	time.Sleep(time.Second * 5)
//
//	srv, err := app.GetRegistry().GetService(reportsv1.ServiceName)
//	if err != nil {
//		app.GetLogger().Fatal(err)
//		return
//	}
//	client := report.GetClient(srv.GetClientConn())
//	getReport, err := client.GetReport(context.Background(), &reportsv1.GetReportRequest{})
//	if err != nil {
//		app.GetLogger().Fatal(err)
//	}
//	app.GetLogger().Println(getReport.GetReport())
//}

func customRouter(app *tonica.App) {
	// adding custom gin handlers without fluent API and automatic OpenAPI documentation
	app.GetRouter().GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"hello": "world",
		})
	})

	// Example with query parameters
	tonica.NewRoute(app).
		GET("/greet").
		Summary("Greet user").
		Description("Returns a personalized greeting").
		Tags("Custom").
		QueryParam("name", "string", "Name of the person to greet", true).
		QueryParam("lang", "string", "Language code (en, ru, etc.)", false).
		Response(200, "Greeting message", tonica.InlineObjectSchema(map[string]string{
			"message": "string",
		})).
		Response(400, "Bad request", tonica.InlineObjectSchema(map[string]string{
			"error": "string",
		})).
		Handle(func(c *gin.Context) {
			name := c.Query("name")
			if name == "" {
				c.JSON(400, gin.H{"error": "name is required"})
				return
			}
			lang := c.DefaultQuery("lang", "en")

			greetings := map[string]string{
				"en": "Hello",
				"ru": "Привет",
				"es": "Hola",
			}

			greeting, ok := greetings[lang]
			if !ok {
				greeting = greetings["en"]
			}

			c.JSON(200, gin.H{
				"message": fmt.Sprintf("%s, %s!", greeting, name),
			})
		})

	// Example with path parameter
	tonica.NewRoute(app).
		GET("/users/:id").
		Summary("Get user by ID").
		Description("Retrieves user information by user ID").
		Tags("Custom", "Users").
		PathParam("id", "string", "User ID").
		Response(200, "User information", tonica.InlineObjectSchema(map[string]string{
			"id":    "string",
			"name":  "string",
			"email": "string",
		})).
		Response(404, "User not found", tonica.InlineObjectSchema(map[string]string{
			"error": "string",
		})).
		Handle(func(c *gin.Context) {
			id := c.Param("id")
			// Mock user data
			c.JSON(200, gin.H{
				"id":    id,
				"name":  "John Doe",
				"email": "john@example.com",
			})
		})

	// Example POST with body
	tonica.NewRoute(app).
		POST("/users").
		Summary("Create new user").
		Description("Creates a new user with the provided data").
		Tags("Custom", "Users").
		BodyParam("User data", tonica.InlineObjectSchema(map[string]string{
			"name":  "string",
			"email": "string",
		})).
		Response(201, "User created successfully", tonica.InlineObjectSchema(map[string]string{
			"id":      "string",
			"message": "string",
		})).
		Response(400, "Invalid input", tonica.InlineObjectSchema(map[string]string{
			"error": "string",
		})).
		Handle(func(c *gin.Context) {
			var body map[string]interface{}
			if err := c.BindJSON(&body); err != nil {
				c.JSON(400, gin.H{"error": "invalid JSON"})
				return
			}

			c.JSON(201, gin.H{
				"id":      "usr_123",
				"message": "User created successfully",
			})
		})
}
