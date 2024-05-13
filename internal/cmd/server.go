package cmd

import (
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/server"
	"github.com/prometheus/common/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/attachmentgenie/atc/pkg/atc"
)

var logLevel string
var port int
var target []string

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start as a background process.",
	Long:  "Start as a background process.",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := atc.Config{
			Server: server.Config{
				HTTPListenPort:   port,
				MetricsNamespace: "atc",
			},
			Target: target,
		}
		_ = cfg.Server.LogLevel.Set(logLevel)
		t, err := atc.New(cfg)
		if err != nil {
			panic(err)
		}

		level.Info(t.Cfg.Server.Log).Log("msg", "Starting application", "version", version.Info())
		err = t.Run()
		if err != nil {
			panic(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.PersistentFlags().IntVar(&port, "port", 8088, "port to expose service on.")
	viper.BindPFlag("port", serverCmd.PersistentFlags().Lookup("port"))
	serverCmd.PersistentFlags().StringSliceVar(&target, "target", []string{"all"}, "Comma-separated list of components to include in the instantiated process. Use the 'modules' command line flag to get a list of available components, and to see which components are included with 'all'. (default all)")
	viper.BindPFlag("target", serverCmd.PersistentFlags().Lookup("target"))
	serverCmd.PersistentFlags().StringVarP(&logLevel, "log_level", "", "info", "Only log messages with the given severity or above.")
	viper.BindPFlag("log_level", serverCmd.PersistentFlags().Lookup("log_level"))
}
