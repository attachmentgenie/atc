package cmd

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	otelcontrib "go.opentelemetry.io/contrib"
	runtimemetrics "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		configureOpentelemetry()

		meter := global.MeterProvider().Meter("example", metric.WithInstrumentationVersion(otelcontrib.Version()))
		counter, err := meter.Int64Counter(
			"test.my_counter",
			metric.WithDescription("Just a test counter"),
		)
		if err != nil {
			panic(err)
		}

		for {
			n := rand.Intn(1000)
			time.Sleep(time.Duration(n) * time.Millisecond)

			counter.Add(ctx, 1)
		}
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serverCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serverCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func configureOpentelemetry() {
	if err := runtimemetrics.Start(); err != nil {
		panic(err)
	}
	_ = configureMetrics()

	http.HandleFunc("/status", ping)
	http.HandleFunc("/ready", ping)
	http.Handle("/metrics", promhttp.Handler())

	go func() {
		_ = http.ListenAndServe(":8088", nil)
	}()
}

func configureMetrics() *prometheus.Exporter {
	exporter, err := prometheus.New()
	if err != nil {
		log.Fatal(err)
	}
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))

	global.SetMeterProvider(provider)

	return exporter
}

func ping(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "pong")
}
