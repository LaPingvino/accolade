//go:build race

package collab

// The race detector slows this package several times over; time limits
// scale with it.
const slowdown = 10
