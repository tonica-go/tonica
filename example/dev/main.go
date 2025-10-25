package main

import (
	"strings"

	paymentv1 "github.com/tonica-go/tonica/example/dev/proto/payment/v1"
	reportsv1 "github.com/tonica-go/tonica/example/dev/proto/reports/v1"
	"github.com/tonica-go/tonica/example/dev/services/payment"
	"github.com/tonica-go/tonica/example/dev/services/report"
	"github.com/tonica-go/tonica/pkg/tonica"
	"github.com/tonica-go/tonica/pkg/tonica/config"
	"github.com/tonica-go/tonica/pkg/tonica/consumer"
	"github.com/tonica-go/tonica/pkg/tonica/service"
	"github.com/tonica-go/tonica/pkg/tonica/storage/pubsub/kafka"
)

func main() {
	app := tonica.NewApp(
		tonica.WithSpec("example/dev/openapi/openapi.swagger.json"),
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

	app.GetRegistry().MustRegisterConsumer(
		consumer.NewConsumer(
			consumer.WithName("payments"),
			consumer.WithClient(kafka.New(&kafka.Config{
				Brokers:         []string{config.GetEnv("KAFKA_BROKERS", "")},
				ConsumerGroupID: config.GetEnv("KAFKA_CONSUMER_GROUP_ID", "payments"),
				BatchBytes:      config.GetEnvInt("KAFKA_BATCH_BYTES", 5000),
				BatchTimeout:    config.GetEnvInt("KAFKA_BATCH_TIMEOUT", 5000),
				BatchSize:       config.GetEnvInt("KAFKA_BATCH_SIZE", 5000),
				TLS:             kafka.TLSConfig{},
			}, app.GetMetricManager())),
			consumer.WithHandler(payment.GetConsumer(paymentSrc)),
		),
	)

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
