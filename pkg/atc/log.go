package atc

import (
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/server"
	"os"
	"time"
)

var (
	DefaultTimestampUTC = log.TimestampFormat(
		func() time.Time { return time.Now().UTC() },
		time.DateTime,
	)
	Logger log.Logger
)

func InitLogger(cfg *server.Config) {
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", DefaultTimestampUTC, "caller", log.Caller(5))

	Logger = level.NewFilter(logger)
	cfg.Log = level.NewFilter(log.With(logger, "caller", log.Caller(6)))
}
