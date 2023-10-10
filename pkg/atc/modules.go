package atc

import (
	"github.com/attachmentgenie/atc/pkg/atc/autoscaler"
	"github.com/attachmentgenie/atc/pkg/atc/deployer"
	"github.com/attachmentgenie/atc/pkg/atc/event_sink"
	"github.com/attachmentgenie/atc/pkg/atc/forwarder"
	"github.com/attachmentgenie/atc/pkg/atc/redirecter"
	"github.com/grafana/dskit/modules"
	"github.com/grafana/dskit/server"
	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus/collectors"
	promversion "github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	"golang.org/x/exp/slices"
)

const (
	API        string = "api"
	Autoscaler string = "autoscaler"
	Consul     string = "consul"
	Deployer   string = "deployer"
	EventSink  string = "event_sink"
	Forwarder  string = "forwarder"
	Nomad      string = "nomad"
	Server     string = "server"
	Redirecter string = "redirecter"
	All        string = "all"
)

func (t *Atc) initAPI() (services.Service, error) {
	landingConfig := web.LandingConfig{
		Name:        "ATC",
		Description: "Atc",
		Version:     promversion.Version,
		Links: []web.LandingLinks{
			{
				Address: "/health",
				Text:    "Health",
			},
			{
				Address: "/metrics",
				Text:    "Metrics",
			},
			{
				Address: "/ready",
				Text:    "Ready",
			},
		},
	}
	landingPage, err := web.NewLandingPage(landingConfig)
	if err != nil {
		panic(err)
	}
	t.Server.HTTP.Handle("/", landingPage)

	return nil, nil
}

func (t *Atc) initAutoscaler() (services.Service, error) {
	autoscaler, err := autoscaler.New(Logger)
	if err != nil {
		return nil, err
	}
	t.Autoscaler = autoscaler
	return t.Autoscaler, nil
}

func (t *Atc) initDeployer() (services.Service, error) {
	//t.Server.HTTP.Handle("/v1/jobs", ah).Methods("GET", "PUT")
	//t.Server.HTTP.Handle("/v1/validate/job", ah).Methods("PUT")

	deploy, err := deployer.New(Logger)
	if err != nil {
		return nil, err
	}
	t.Deployer = deploy
	return t.Deployer, nil
}

func (t *Atc) initEventSink() (services.Service, error) {
	sink, err := event_sink.New(Logger)
	if err != nil {
		return nil, err
	}
	t.EventSink = sink
	return t.EventSink, nil
}

func (t *Atc) initForwarder() (services.Service, error) {
	forward, err := forwarder.New(Logger)
	if err != nil {
		return nil, err
	}
	t.Forwarder = forward
	return t.Forwarder, nil
}

func (t *Atc) initServer() (services.Service, error) {

	t.Cfg.Server.RegisterInstrumentation = true
	DisableSignalHandling(&t.Cfg.Server)

	serv, err := server.New(t.Cfg.Server)
	if err != nil {
		return nil, err
	}
	serv.Registerer.Unregister(collectors.NewGoCollector())
	serv.Registerer.MustRegister(promversion.NewCollector(t.Cfg.Server.MetricsNamespace))

	t.Server = serv

	servicesToWaitFor := func() []services.Service {
		svs := []services.Service(nil)

		serverDeps := t.ModuleManager.DependenciesForModule(Server)

		for m, s := range t.ServiceMap {
			// Server should not wait for itself or for any of its dependencies.
			if m == Server {
				continue
			}

			if slices.Contains(serverDeps, m) {
				continue
			}

			svs = append(svs, s)
		}
		return svs
	}

	s := NewServerService(t.Server, servicesToWaitFor)

	return s, nil
}

func (t *Atc) initRedirecter() (services.Service, error) {
	redirect, err := redirecter.New(Logger)
	if err != nil {
		return nil, err
	}
	t.Redirecter = redirect
	return t.Redirecter, nil
}

func (t *Atc) setupModuleManager() error {
	mm := modules.NewManager(Logger)
	mm.RegisterModule(Server, t.initServer, modules.UserInvisibleModule)
	mm.RegisterModule(API, t.initAPI, modules.UserInvisibleModule)
	mm.RegisterModule(Autoscaler, t.initAutoscaler)
	mm.RegisterModule(Deployer, t.initDeployer)
	mm.RegisterModule(EventSink, t.initEventSink)
	mm.RegisterModule(Forwarder, t.initForwarder)
	mm.RegisterModule(Redirecter, t.initRedirecter)
	mm.RegisterModule(Consul, nil)
	mm.RegisterModule(Nomad, nil)
	mm.RegisterModule(All, nil)

	deps := map[string][]string{
		API:        {Server},
		Autoscaler: {Server},
		Consul:     {Forwarder, Redirecter},
		Deployer:   {API},
		EventSink:  {Server},
		Nomad:      {Deployer, EventSink},
		All:        {API, Deployer, EventSink, Forwarder, Redirecter},
	}
	for mod, targets := range deps {
		if err := mm.AddDependency(mod, targets...); err != nil {
			return err
		}
	}
	t.ModuleManager = mm

	return nil
}
