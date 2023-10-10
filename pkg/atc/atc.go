package atc

import (
	"context"
	"fmt"
	"github.com/attachmentgenie/atc/pkg/atc/autoscaler"
	"github.com/attachmentgenie/atc/pkg/atc/deployer"
	"github.com/attachmentgenie/atc/pkg/atc/event_sink"
	"github.com/attachmentgenie/atc/pkg/atc/redirecter"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/dskit/modules"
	"github.com/grafana/dskit/server"
	"github.com/grafana/dskit/services"
	"github.com/grafana/dskit/signals"
	"github.com/pkg/errors"
	"go.uber.org/atomic"
	"net/http"
	"strings"

	"github.com/attachmentgenie/atc/pkg/atc/forwarder"
)

type Config struct {
	Name   string                 `yaml:"service"`
	Server server.Config          `yaml:"server"`
	Target flagext.StringSliceCSV `yaml:"target"`
}

type Atc struct {
	Cfg Config

	Server *server.Server

	Autoscaler *autoscaler.Autoscaler
	Deployer   *deployer.Deployer
	EventSink  *event_sink.EventSink
	Forwarder  *forwarder.Forwarder
	Redirecter *redirecter.Redirecter

	// set during initialization
	ServiceMap    map[string]services.Service
	ModuleManager *modules.Manager
}

func New(cfg Config) (*Atc, error) {
	InitLogger(&cfg.Server)

	atc := &Atc{
		Cfg: cfg,
	}

	if err := atc.setupModuleManager(); err != nil {
		return nil, err
	}

	return atc, nil
}

func (t *Atc) Run() error {
	for _, module := range t.Cfg.Target {
		if !t.ModuleManager.IsTargetableModule(module) {
			return fmt.Errorf("selected target (%s) is an internal module, which is not allowed", module)
		}
	}

	var err error
	t.ServiceMap, err = t.ModuleManager.InitModuleServices(t.Cfg.Target...)
	if err != nil {
		return err
	}

	// get all services, create service manager and tell it to start
	servs := []services.Service(nil)
	for _, s := range t.ServiceMap {
		servs = append(servs, s)
	}

	sm, err := services.NewManager(servs...)
	if err != nil {
		return err
	}

	// Used to delay shutdown but return "not ready" during this delay.
	shutdownRequested := atomic.NewBool(false)

	t.Server.HTTP.Path("/ready").Handler(t.readyHandler(sm, shutdownRequested))
	t.Server.HTTP.Path("/health").Handler(t.readyHandler(sm, shutdownRequested))

	// Let's listen for events from this manager, and log them.
	healthy := func() { level.Info(Logger).Log("msg", "Application started") }
	stopped := func() { level.Info(Logger).Log("msg", "Application stopped") }
	serviceFailed := func(service services.Service) {
		// if any service fails, stop entire Mimir
		sm.StopAsync()

		// let's find out which module failed
		for m, s := range t.ServiceMap {
			if s == service {
				if errors.Is(service.FailureCase(), modules.ErrStopProcess) {
					level.Info(Logger).Log("msg", "received stop signal via return error", "module", m, "err", service.FailureCase())
				} else {
					level.Error(Logger).Log("msg", "module failed", "module", m, "err", service.FailureCase())
				}
				return
			}
		}

		level.Error(Logger).Log("msg", "module failed", "module", "unknown", "err", service.FailureCase())
	}

	sm.AddListener(services.NewManagerListener(healthy, stopped, serviceFailed))

	handler := signals.NewHandler(t.Server.Log)
	go func() {
		handler.Loop()

		shutdownRequested.Store(true)
		t.Server.HTTPServer.SetKeepAlivesEnabled(false)

		sm.StopAsync()
	}()

	// Start all services. This can really only fail if some service is already
	// in other state than New, which should not be the case.
	err = sm.StartAsync(context.Background())
	if err == nil {
		// Wait until service manager stops. It can stop in two ways:
		// 1) Signal is received and manager is stopped.
		// 2) Any service fails.
		err = sm.AwaitStopped(context.Background())
	}

	// If there is no error yet (= service manager started and then stopped without problems),
	// but any service failed, report that failure as an error to caller.
	if err == nil {
		if failed := sm.ServicesByState()[services.Failed]; len(failed) > 0 {
			for _, f := range failed {
				if !errors.Is(f.FailureCase(), modules.ErrStopProcess) {
					// Details were reported via failure listener before
					err = errors.New("failed services")
					break
				}
			}
		}
	}
	return err
}

func (t *Atc) readyHandler(sm *services.Manager, shutdownRequested *atomic.Bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if shutdownRequested.Load() {
			level.Debug(Logger).Log("msg", "application is stopping")
			http.Error(w, "Application is stopping", http.StatusServiceUnavailable)
			return
		}

		if !sm.IsHealthy() {
			var serviceNamesStates []string
			for name, s := range t.ServiceMap {
				if s.State() != services.Running {
					serviceNamesStates = append(serviceNamesStates, fmt.Sprintf("%s: %s", name, s.State()))
				}
			}

			level.Debug(Logger).Log("msg", "some services are not Running", "services", strings.Join(serviceNamesStates, ", "))
			httpResponse := "Some services are not Running:\n" + strings.Join(serviceNamesStates, "\n")
			http.Error(w, httpResponse, http.StatusServiceUnavailable)
			return
		}
		fmt.Fprintf(w, "OK")
	}
}
