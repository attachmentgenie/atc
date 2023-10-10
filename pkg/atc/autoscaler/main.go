package autoscaler

import (
	"context"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/services"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"time"
)

type Autoscaler struct {
	services.Service

	logger log.Logger
}

func (es *Autoscaler) starting(ctx context.Context) error {
	return nil
}

func (es *Autoscaler) stopping(_ error) error {
	return nil
}

func New(logger log.Logger) (*Autoscaler, error) {

	es := &Autoscaler{
		logger: logger,
	}
	es.Service = services.NewBasicService(es.starting, es.watcher, es.stopping)
	return es, nil
}

func (es *Autoscaler) watcher(ctx context.Context) error {
	client, err := api.NewClient(&api.Config{})
	if err != nil {
		return err
	}

	for {
		watcher, parseErr := watch.Parse(map[string]interface{}{"type": "services"})
		if parseErr != nil {
			level.Error(es.logger).Log("msg", "failed to create services watcher plan: %s", parseErr.Error())
		}
		watcher.HybridHandler = func(_ watch.BlockingParamVal, _ interface{}) {
			level.Debug(es.logger).Log("msg", "Consul handler fired")
		}
		watcherErr := watcher.RunWithClientAndHclog(client, watcher.Logger)
		if watcherErr != nil {
			return watcherErr
		}
		time.Sleep(1 * time.Second)
	}
}
