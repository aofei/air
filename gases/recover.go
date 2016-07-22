package gases

import (
	"fmt"
	"runtime"

	"air"
)

type (
	// RecoverConfig defines the config for recover gas.
	RecoverConfig struct {
		// Size of the stack to be printed.
		// Optional. Default value 4KB.
		StackSize int `json:"stack_size"`

		// DisableStackAll disables formatting stack traces of all other goroutines
		// into buffer after the trace for the current goroutine.
		// Optional. Default value false.
		DisableStackAll bool `json:"disable_stack_all"`

		// DisablePrintStack disables printing stack trace.
		// Optional. Default value as false.
		DisablePrintStack bool `json:"disable_print_stack"`
	}
)

var (
	// DefaultRecoverConfig is the default recover gas config.
	DefaultRecoverConfig = RecoverConfig{
		StackSize:         4 << 10, // 4 KB
		DisableStackAll:   false,
		DisablePrintStack: false,
	}
)

// Recover returns a gas which recovers from panics anywhere in the chain
// and handles the control to the centralized HTTPErrorHandler.
func Recover() air.GasFunc {
	return RecoverWithConfig(DefaultRecoverConfig)
}

// RecoverWithConfig returns a recover gas from config.
// See: `Recover()`.
func RecoverWithConfig(config RecoverConfig) air.GasFunc {
	// Defaults
	if config.StackSize == 0 {
		config.StackSize = DefaultRecoverConfig.StackSize
	}

	return func(next air.HandlerFunc) air.HandlerFunc {
		return func(c air.Context) error {
			defer func() {
				if r := recover(); r != nil {
					var err error
					switch r := r.(type) {
					case error:
						err = r
					default:
						err = fmt.Errorf("%v", r)
					}
					stack := make([]byte, config.StackSize)
					length := runtime.Stack(stack, !config.DisableStackAll)
					if !config.DisablePrintStack {
						c.Logger().Printf("[%s] %s %s", "PANIC RECOVER", err, stack[:length])
					}
					c.Error(err)
				}
			}()
			return next(c)
		}
	}
}
