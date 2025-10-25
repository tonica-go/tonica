package main

import (
	"strings"

	paymentv1 "github.com/alexrett/tonica/example/dev/proto/payment/v1"
	reportsv1 "github.com/alexrett/tonica/example/dev/proto/reports/v1"
	"github.com/alexrett/tonica/example/dev/services/payment"
	"github.com/alexrett/tonica/example/dev/services/report"
	"github.com/alexrett/tonica/pkg/tonica"
	"github.com/alexrett/tonica/pkg/tonica/config"
	"github.com/alexrett/tonica/pkg/tonica/service"
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
	app.GetRegistry().MustRegisterService(
		service.NewService(
			service.WithName(paymentv1.ServiceName),
			service.WithGRPC(payment.RegisterGRPC),
			service.WithGRPCAddr(config.GetEnv(paymentv1.ServiceAddrEnvName, ":9000")),
			service.WithGateway(payment.RegisterGateway),
		),
	)

	app.GetRegistry().MustRegisterService(
		service.NewService(
			service.WithName(reportsv1.ServiceName),
			service.WithGRPC(report.RegisterGRPC),
			service.WithGRPCAddr(config.GetEnv(reportsv1.ServiceAddrEnvName, ":9001")),
		),
	)
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
