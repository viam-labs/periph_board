//go:build linux

// Package periphboard implements a Linux-based board which uses periph.io for GPIO pins.
package periphboard

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	pb "go.viam.com/api/component/board/v1"
	goutils "go.viam.com/utils"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/host/v3"

	"go.viam.com/rdk/components/board"
	"go.viam.com/rdk/grpc"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
)

var Model = resource.NewModel("viam-labs", "board", "periph")

func init() {
	resource.RegisterComponent(
		board.API,
		Model,
		resource.Registration[board.Board, *resource.NoNativeConfig]{Constructor: newBoard})
}

func newBoard(
	ctx context.Context,
	_ resource.Dependencies,
	conf resource.Config,
	logger logging.Logger,
) (board.Board, error) {
	if _, err := host.Init(); err != nil {
		logger.Warnf("error initializing periph host", "error", err)
	}

	cancelCtx, cancelFunc := context.WithCancel(context.Background())
	b := sysfsBoard{
		Named:      conf.ResourceName().AsNamed(),
		logger:     logger,
		cancelCtx:  cancelCtx,
		cancelFunc: cancelFunc,

		// this is not yet modified during reconfiguration but maybe should be
		pwms: map[string]pwmSetting{},
	}

	return &b, nil
}

type sysfsBoard struct {
	resource.Named
	resource.TriviallyReconfigurable

	mu      sync.RWMutex
	pwms    map[string]pwmSetting
	logger  logging.Logger

	cancelCtx               context.Context
	cancelFunc              func()
	activeBackgroundWorkers sync.WaitGroup
}

type pwmSetting struct {
	dutyCycle gpio.Duty
	frequency physic.Frequency
}

func (b *sysfsBoard) AnalogByName(name string) (board.Analog, error) {
	return nil, grpc.UnimplementedError
}

func (b *sysfsBoard) DigitalInterruptByName(name string) (board.DigitalInterrupt, error) {
	return nil, grpc.UnimplementedError
}

func (b *sysfsBoard) AnalogNames() []string {
	return nil
}

func (b *sysfsBoard) DigitalInterruptNames() []string {
	return nil
}

func (b *sysfsBoard) getGPIOLine(hwPin string) (gpio.PinIO, bool, error) {
	pinName := hwPin
	hwPWMSupported := false

	pin := gpioreg.ByName(pinName)
	if pin == nil {
		return nil, false, errors.Errorf("no global pin found for %q", pinName)
	}
	return pin, hwPWMSupported, nil
}

func (b *sysfsBoard) GPIOPinByName(pinName string) (board.GPIOPin, error) {
	pin, hwPWMSupported, err := b.getGPIOLine(pinName)
	if err != nil {
		return nil, err
	}

	return periphGpioPin{b, pin, pinName, hwPWMSupported}, nil
}

func (b *sysfsBoard) WriteAnalog(ctx context.Context, pin string, value int32, extra map[string]interface{}) error {
	return grpc.UnimplementedError
}

// expects to already have lock acquired.
func (b *sysfsBoard) startSoftwarePWMLoop(gp periphGpioPin) {
	b.activeBackgroundWorkers.Add(1)
	goutils.ManagedGo(func() {
		b.softwarePWMLoop(b.cancelCtx, gp)
	}, b.activeBackgroundWorkers.Done)
}

func (b *sysfsBoard) softwarePWMLoop(ctx context.Context, gp periphGpioPin) {
	for {
		cont := func() bool {
			b.mu.RLock()
			defer b.mu.RUnlock()
			pwmSetting, ok := b.pwms[gp.pinName]
			if !ok {
				b.logger.Debug("pwm setting deleted; stopping")
				return false
			}

			if err := gp.set(true); err != nil {
				b.logger.Errorw("error setting pin", "pin_name", gp.pinName, "error", err)
				return true
			}
			onPeriod := time.Duration(
				int64((float64(pwmSetting.dutyCycle) / float64(gpio.DutyMax)) * float64(pwmSetting.frequency.Period())),
			)
			if !goutils.SelectContextOrWait(ctx, onPeriod) {
				return false
			}
			if err := gp.set(false); err != nil {
				b.logger.Errorw("error setting pin", "pin_name", gp.pinName, "error", err)
				return true
			}
			offPeriod := pwmSetting.frequency.Period() - onPeriod

			return goutils.SelectContextOrWait(ctx, offPeriod)
		}()
		if !cont {
			return
		}
	}
}

func (b *sysfsBoard) SetPowerMode(ctx context.Context, mode pb.PowerMode, duration *time.Duration) error {
	return grpc.UnimplementedError
}

func (b *sysfsBoard) StreamTicks(
	ctx context.Context,
	interrupts []board.DigitalInterrupt,
	ch chan board.Tick,
	extra map[string]interface{},
) error {
	return grpc.UnimplementedError
}

func (b *sysfsBoard) Close(ctx context.Context) error {
	b.mu.Lock()
	b.cancelFunc()
	b.mu.Unlock()
	b.activeBackgroundWorkers.Wait()
	return nil
}
