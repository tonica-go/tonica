package tonica

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/tonica-go/tonica/pkg/tonica/config"
)

func (a *App) Run() error {
	// Implementation of the Run method goes here.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Observability
	o, err := initObs(ctx, a.cfg.AppName())
	if err != nil {
		a.logger.Fatal(err)
	}
	defer func() { _ = o.Shutdown(context.Background()) }()

	switch a.cfg.GetRunMode() {
	case config.ModeAIO:
		a.runAio(ctx)
	case config.ModeService:
		a.runService(ctx)
	case config.ModeWorker:
		a.runWorker(ctx)
	case config.ModeConsumer:
		a.runConsumer(ctx)
	case config.ModeGateway:
		a.runGateway(ctx)
	default:
		return nil
	}
	return nil
}
