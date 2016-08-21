package gases

import "github.com/sheng/air"

// Skipper defines a function to skip gas. Returning true skips processing
// the gas.
type Skipper func(c *air.Context) bool

// defaultSkipper returns false which processes the gas.
func defaultSkipper(c *air.Context) bool {
	return false
}
