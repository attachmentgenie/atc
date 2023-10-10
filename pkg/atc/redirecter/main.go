package redirecter

import (
	"context"
	"fmt"
	"github.com/go-kit/log"
	"github.com/grafana/dskit/services"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
)

type Redirecter struct {
	services.Service

	logger log.Logger

	watchServicesChan chan struct{}
}

func (f *Redirecter) starting(ctx context.Context) error {
	return nil
}

func (f *Redirecter) stopping(_ error) error {
	return nil
}

func New(logger log.Logger) (*Redirecter, error) {

	f := &Redirecter{
		logger: logger,
	}
	f.Service = services.NewBasicService(f.starting, f.watcher, f.stopping)
	return f, nil
}

func (f *Redirecter) watcher(ctx context.Context) error {
	client, err := api.NewClient(&api.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to nomad: %s", err.Error())
	}

	servicesWatcher, parseErr := watch.Parse(map[string]interface{}{"type": "services"})
	if parseErr != nil {
		return fmt.Errorf("failed to create services watcher plan: %s", parseErr.Error())
	}

	servicesWatcher.HybridHandler = func(_ watch.BlockingParamVal, _ interface{}) {
		select {
		case <-ctx.Done():
		case f.watchServicesChan <- struct{}{}:
		default:
			// Event chan is full, discard event.
		}
	}

	checksWatcher, err := watch.Parse(map[string]interface{}{"type": "checks"})
	if err != nil {
		return fmt.Errorf("failed to create checks watcher plan: %w", err)
	}

	checksWatcher.HybridHandler = func(_ watch.BlockingParamVal, _ interface{}) {
		select {
		case <-ctx.Done():
		case f.watchServicesChan <- struct{}{}:
		default:
			// Event chan is full, discard event.
		}
	}

	errChan := make(chan error, 2)

	defer func() {
		servicesWatcher.Stop()
		checksWatcher.Stop()
	}()

	go func() {
		errChan <- servicesWatcher.RunWithClientAndHclog(client, servicesWatcher.Logger)
	}()

	go func() {
		errChan <- checksWatcher.RunWithClientAndHclog(client, checksWatcher.Logger)
	}()

	select {
	case <-ctx.Done():
		return nil

	case err = <-errChan:
		return fmt.Errorf("services or checks watcher terminated: %w", err)
	}
}
