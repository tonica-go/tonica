package tonica

import "github.com/tonica-go/tonica/pkg/tonica/config"

func (a *App) Run() error {
	// Implementation of the Run method goes here.
	switch a.cfg.GetRunMode() {
	case config.ModeAIO:
		a.runAio()
	case config.ModeService:
		a.runService()
	case config.ModeWorker:
	case config.ModeConsumer:
	default:
		return nil
	}
	return nil
}
