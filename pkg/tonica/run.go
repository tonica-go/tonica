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
		a.runAio(ctx, o)
	case config.ModeService:
		a.runService(ctx, o)
	case config.ModeWorker:
		a.runWorker(ctx, o)
	case config.ModeConsumer:
		a.runConsumer(ctx, o)
	case config.ModeGateway:
		a.runGateway(ctx, o)
	default:
		return nil
	}
	return nil
}
