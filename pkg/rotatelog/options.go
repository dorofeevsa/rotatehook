package rotatelog

import (
	"rotateloghook/pkg/rotatelog/internal/option"
	"time"
)

const (
	OptKeyClock         = "clock"
	OptKeyLinkName      = "link-name"
	OptKeyMaxAge        = "max-age"
	OptKeyRotationTime  = "rotation-time"
	OptKeyRotationCount = "rotation-count"
	OptKeyRotationSize  = "rotation-size"
)

// WithClock creates a new Option that sets a clock
// that the RotateLogs object will use to determine
// the current time.
//
// By default rotatelogs.Local, which returns the
// current time in the local time zone, is used. If you
// would rather use UTC, use rotatelogs.UTC as the argument
// to this option, and pass it to the constructor.
func WithClock(c Clock) Option {
	return option.New(OptKeyClock, c)
}

// WithLocation creates a new Option that sets up a
// "Clock" interface that the RotateLogs object will use
// to determine the current time.
//
// This optin works by always returning the in the given
// location.
func WithLocation(loc *time.Location) Option {
	return option.New(OptKeyClock, clockFn(func() time.Time {
		return time.Now().In(loc)
	}))
}

// WithLinkName creates a new Option that sets the
// symbolic link name that gets linked to the current
// file name being used.
func WithLinkName(s string) Option {
	return option.New(OptKeyLinkName, s)
}

// WithMaxAge creates a new Option that sets the
// max age of a log file before it gets purged from
// the file system.
func WithMaxAge(d time.Duration) Option {
	return option.New(OptKeyMaxAge, d)
}

// WithRotationTime creates a new Option that sets the
// time between rotation.
func WithRotationTime(d time.Duration) Option {
	return option.New(OptKeyRotationTime, d)
}

// WithRotationCount creates a new Option that sets the
// number of files should be kept before it gets
// purged from the file system.
func WithRotationCount(n int) Option {
	return option.New(OptKeyRotationCount, n)
}

func WithRotationSize(s int64) Option {
	return option.New(OptKeyRotationSize, s)
}
