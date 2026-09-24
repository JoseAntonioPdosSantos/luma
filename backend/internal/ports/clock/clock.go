// Package clock provides an injectable source of the current time so that
// application services can be tested deterministically.
package clock

import "time"

// Clock returns the current time in UTC.
type Clock interface {
	Now() time.Time
}

// Real is a Clock backed by the system clock.
type Real struct{}

func (Real) Now() time.Time { return time.Now().UTC() }

// Fixed is a Clock that always returns the same instant. Useful in tests.
type Fixed struct {
	Time time.Time
}

func (f Fixed) Now() time.Time { return f.Time }
