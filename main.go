package main

import (
	"context"
	
	"github.com/edaniels/golog"
	goutils "go.viam.com/utils"
	
	"go.viam.com/rdk/components/board"
	"go.viam.com/rdk/module"
	"go.viam.com/rdk/resource"
	
	"github.com/viam-labs/periph_board/periphboard"
)

// This should be considered const, but go doesn't let you build const things from function calls.
// modelName is in fake_sensor.go for now.
var model = resource.NewModel("viamlabs", "board", periphboard.ModelName)

func mainWithArgs(ctx context.Context, args []string, logger golog.Logger) (err error) {
	modalModule, err := module.NewModuleFromArgs(ctx, logger)
	if err != nil {
		return err
	}
	modalModule.AddModelFromRegistry(ctx, board.API, model)

	err = modalModule.Start(ctx)
	defer modalModule.Close(ctx)
	if err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}

func main() {
	goutils.ContextualMain(mainWithArgs, golog.NewDevelopmentLogger("periphboard"))
}
