//go:build race

package pace

// The race detector slows every call several times over; time limits
// scale with it.
const slowdown = 10
