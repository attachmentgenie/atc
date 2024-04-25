package atc

import (
	"github.com/grafana/dskit/modules"
	"github.com/grafana/dskit/server"
	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus/collectors"
	promversion "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	"golang.org/x/exp/slices"

	"github.com/attachmentgenie/atc/pkg/atc/autoscaler"
	"github.com/attachmentgenie/atc/pkg/atc/deployer"
	"github.com/attachmentgenie/atc/pkg/atc/event_sink"
	"github.com/attachmentgenie/atc/pkg/atc/forwarder"
	"github.com/attachmentgenie/atc/pkg/atc/incident"
	"github.com/attachmentgenie/atc/pkg/atc/radar"
	"github.com/attachmentgenie/atc/pkg/atc/redirecter"
)

const (
	API        string = "api"
	Autoscaler string = "autoscaler"
	Boundary   string = "boundary"
	Consul     string = "consul"
	Deployer   string = "deployer"
	EventSink  string = "event_sink"
	Forwarder  string = "forwarder"
	Incident   string = "incident"
	Nomad      string = "nomad"
	Server     string = "server"
	Radar      string = "radar"
	Redirecter string = "redirecter"
	All        string = "all"
)

func (t *Atc) initAPI() (services.Service, error) {
	landingConfig := web.LandingConfig{
		Name:        "ATC",
		Description: "Atc",
		Version:     version.Version,
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
			{
				Address: "/services",
				Text:    "Services",
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
	autosclr, err := autoscaler.New(t.logger)
	if err != nil {
		return nil, err
	}
	t.Autoscaler = autosclr
	return t.Autoscaler, nil
}

func (t *Atc) initDeployer() (services.Service, error) {
	deploy, err := deployer.New(t.logger)
	if err != nil {
		return nil, err
	}

	t.Server.HTTP.Path("/v1/jobs").Methods("GET", "PUT").Handler(deploy.OkHandler())
	t.Server.HTTP.Path("/v1/validate/job").Methods("PUT").Handler(deploy.OkHandler())

	t.Deployer = deploy
	return t.Deployer, nil
}

func (t *Atc) initEventSink() (services.Service, error) {
	sink, err := event_sink.New(t.logger)
	if err != nil {
		return nil, err
	}
	t.EventSink = sink
	return t.EventSink, nil
}

func (t *Atc) initForwarder() (services.Service, error) {
	forward, err := forwarder.New(t.logger)
	if err != nil {
		return nil, err
	}
	t.Forwarder = forward
	return t.Forwarder, nil
}

func (t *Atc) initIncident() (services.Service, error) {
	incident, err := incident.New(t.logger)
	if err != nil {
		return nil, err
	}
	t.Incident = incident
	return t.Incident, nil
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
	serv.HTTP.Path("/ready").Handler(OkHandler())

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

func (t *Atc) initRadar() (services.Service, error) {
	rdr, err := radar.New(t.logger)
	if err != nil {
		return nil, err
	}
	t.Radar = rdr
	return t.Radar, nil
}

func (t *Atc) initRedirecter() (services.Service, error) {
	redirect, err := redirecter.New(t.logger)
	if err != nil {
		return nil, err
	}
	t.Redirecter = redirect
	return t.Redirecter, nil
}

func (t *Atc) setupModuleManager() error {
	mm := modules.NewManager(t.logger)
	mm.RegisterModule(Server, t.initServer, modules.UserInvisibleModule)
	mm.RegisterModule(API, t.initAPI, modules.UserInvisibleModule)
	mm.RegisterModule(Autoscaler, t.initAutoscaler)
	mm.RegisterModule(Deployer, t.initDeployer)
	mm.RegisterModule(EventSink, t.initEventSink)
	mm.RegisterModule(Forwarder, t.initForwarder)
	mm.RegisterModule(Incident, t.initIncident)
	mm.RegisterModule(Radar, t.initRadar)
	mm.RegisterModule(Redirecter, t.initRedirecter)
	mm.RegisterModule(Boundary, nil)
	mm.RegisterModule(Consul, nil)
	mm.RegisterModule(Nomad, nil)
	mm.RegisterModule(All, nil)

	deps := map[string][]string{
		API:      {Server},
		Boundary: {Incident},
		Consul:   {Forwarder, Redirecter},
		Deployer: {API},
		Nomad:    {Autoscaler, Deployer, EventSink},
		All:      {Boundary, Consul, Nomad},
	}
	for mod, targets := range deps {
		if err := mm.AddDependency(mod, targets...); err != nil {
			return err
		}
	}
	t.ModuleManager = mm

	return nil
}
