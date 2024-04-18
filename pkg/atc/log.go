package atc

import (
	"os"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	dslog "github.com/grafana/dskit/log"
)

func initLogger(logFormat string, logLevel dslog.Level) log.Logger {
	writer := log.NewSyncWriter(os.Stderr)
	logger := dslog.NewGoKitWithWriter(logFormat, writer)

	// use UTC timestamps and skip 5 stack frames.
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.Caller(5))
	logger = level.NewFilter(logger, logLevel.Option)

	return logger
}
