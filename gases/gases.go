package gases

import "github.com/sheng/air"

type (
	// Skipper defines a function to skip gas. Returning true skips processing
	// the gas.
	Skipper func(c *air.Context) bool
)

// defaultSkipper returns false which processes the gas.
func defaultSkipper(c *air.Context) bool {
	return false
}
