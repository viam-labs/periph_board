//go:build linux

// Package periphboard implements a Linux-based board which uses periph.io for GPIO pins.
package periphboard

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/edaniels/golog"
	"github.com/pkg/errors"
	commonpb "go.viam.com/api/common/v1"
	pb "go.viam.com/api/component/board/v1"
	goutils "go.viam.com/utils"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/host/v3"

	"go.viam.com/rdk/components/board"
	"go.viam.com/rdk/grpc"
	"go.viam.com/rdk/resource"
)

var Model = resource.NewModel("viam-labs", "board", "periph")

func init() {
	resource.RegisterComponent(
		board.API,
		Model,
		resource.Registration[board.Board, *Config]{Constructor: newBoard})
}

func newBoard(
	ctx context.Context,
	_ resource.Dependencies,
	conf resource.Config,
	logger golog.Logger,
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

		spis:    map[string]*spiBus{},
		analogs: map[string]*wrappedAnalog{},
		// this is not yet modified during reconfiguration but maybe should be
		pwms: map[string]pwmSetting{},
	}

	if err := b.Reconfigure(ctx, nil, conf); err != nil {
		return nil, err
	}
	return &b, nil
}

func (b *sysfsBoard) Reconfigure(
	ctx context.Context,
	_ resource.Dependencies,
	conf resource.Config,
) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	newConf, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		return err
	}

	if err := b.reconfigureSpis(newConf); err != nil {
		return err
	}

	if err := b.reconfigureAnalogs(ctx, newConf); err != nil {
		return err
	}
	return nil
}

// This never returns errors, but we give it the same function signature as the other
// reconfiguration helpers for consistency.
func (b *sysfsBoard) reconfigureSpis(newConf *Config) error {
	stillExists := map[string]struct{}{}
	for _, c := range newConf.SPIs {
		stillExists[c.Name] = struct{}{}
		if curr, ok := b.spis[c.Name]; ok {
			if busPtr := curr.bus.Load(); busPtr != nil && *busPtr != c.BusSelect {
				curr.reset(c.BusSelect)
			}
			continue
		}
		b.spis[c.Name] = &spiBus{}
		b.spis[c.Name].reset(c.BusSelect)
	}

	for name := range b.spis {
		if _, ok := stillExists[name]; ok {
			continue
		}
		delete(b.spis, name)
	}
	return nil
}

func (b *sysfsBoard) reconfigureAnalogs(ctx context.Context, newConf *Config) error {
	stillExists := map[string]struct{}{}
	for _, c := range newConf.Analogs {
		channel, err := strconv.Atoi(c.Pin)
		if err != nil {
			return errors.Errorf("bad analog pin (%s)", c.Pin)
		}

		bus, ok := b.spis[c.SPIBus]
		if !ok {
			return errors.Errorf("can't find SPI bus (%s) requested by AnalogReader", c.SPIBus)
		}

		stillExists[c.Name] = struct{}{}
		if curr, ok := b.analogs[c.Name]; ok {
			if curr.chipSelect != c.ChipSelect {
				ar := &board.MCP3008AnalogReader{channel, bus, c.ChipSelect}
				curr.reset(ctx, curr.chipSelect, board.SmoothAnalogReader(ar, c, b.logger))
			}
			continue
		}
		ar := &board.MCP3008AnalogReader{channel, bus, c.ChipSelect}
		b.analogs[c.Name] = newWrappedAnalog(ctx, c.ChipSelect, board.SmoothAnalogReader(ar, c, b.logger))
	}

	for name := range b.analogs {
		if _, ok := stillExists[name]; ok {
			continue
		}
		b.analogs[name].reset(ctx, "", nil)
		delete(b.analogs, name)
	}
	return nil
}

type wrappedAnalog struct {
	mu         sync.RWMutex
	chipSelect string
	reader     *board.AnalogSmoother
}

func newWrappedAnalog(ctx context.Context, chipSelect string, reader *board.AnalogSmoother) *wrappedAnalog {
	var wrapped wrappedAnalog
	wrapped.reset(ctx, chipSelect, reader)
	return &wrapped
}

func (a *wrappedAnalog) Read(ctx context.Context, extra map[string]interface{}) (int, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.reader == nil {
		return 0, errors.New("closed")
	}
	return a.reader.Read(ctx, extra)
}

func (a *wrappedAnalog) Close(ctx context.Context) error {
	return nil
}

func (a *wrappedAnalog) reset(ctx context.Context, chipSelect string, reader *board.AnalogSmoother) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.reader != nil {
		goutils.UncheckedError(a.reader.Close(ctx))
	}
	a.reader = reader
	a.chipSelect = chipSelect
}

type sysfsBoard struct {
	resource.Named
	mu      sync.RWMutex
	spis    map[string]*spiBus
	analogs map[string]*wrappedAnalog
	pwms    map[string]pwmSetting
	logger  golog.Logger

	cancelCtx               context.Context
	cancelFunc              func()
	activeBackgroundWorkers sync.WaitGroup
}

type pwmSetting struct {
	dutyCycle gpio.Duty
	frequency physic.Frequency
}

func (b *sysfsBoard) SPIByName(name string) (board.SPI, bool) {
	s, ok := b.spis[name]
	return s, ok
}

func (b *sysfsBoard) I2CByName(name string) (board.I2C, bool) {
	return nil, false
}

func (b *sysfsBoard) AnalogReaderByName(name string) (board.AnalogReader, bool) {
	a, ok := b.analogs[name]
	return a, ok
}

func (b *sysfsBoard) DigitalInterruptByName(name string) (board.DigitalInterrupt, bool) {
	return nil, false // Digital interrupts aren't supported.
}

func (b *sysfsBoard) SPINames() []string {
	if len(b.spis) == 0 {
		return nil
	}
	names := make([]string, 0, len(b.spis))
	for k := range b.spis {
		names = append(names, k)
	}
	return names
}

func (b *sysfsBoard) I2CNames() []string {
	return nil
}

func (b *sysfsBoard) AnalogReaderNames() []string {
	names := []string{}
	for k := range b.analogs {
		names = append(names, k)
	}
	return names
}

func (b *sysfsBoard) DigitalInterruptNames() []string {
	return nil
}

func (b *sysfsBoard) GPIOPinNames() []string {
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

func (b *sysfsBoard) Status(ctx context.Context, extra map[string]interface{}) (*commonpb.BoardStatus, error) {
	return board.CreateStatus(ctx, b, extra)
}

func (b *sysfsBoard) ModelAttributes() board.ModelAttributes {
	return board.ModelAttributes{}
}

func (b *sysfsBoard) SetPowerMode(ctx context.Context, mode pb.PowerMode, duration *time.Duration) error {
	return grpc.UnimplementedError
}

func (b *sysfsBoard) Close(ctx context.Context) error {
	b.mu.Lock()
	b.cancelFunc()
	b.mu.Unlock()
	b.activeBackgroundWorkers.Wait()
	return nil
}
