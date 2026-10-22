package generator

import (
	"fmt"
	"runtime"
)

type (
	// GenerateOption configures resource use for one Generate invocation.
	GenerateOption   func(*generationConfig) error
	generationConfig struct{ workers int }
)

// WithRenderWorkers sets a positive maximum number of simultaneous file renders.
// It changes neither process-wide GOMAXPROCS nor analysis/plugin concurrency.
// Without an override, Generate uses min(4, GOMAXPROCS(0)); the last override wins.
func WithRenderWorkers(workers int) GenerateOption {
	return func(config *generationConfig) error {
		if workers < 1 {
			return fmt.Errorf("render workers must be positive, got %d", workers)
		}
		config.workers = workers
		return nil
	}
}

func generationConfiguration(options []GenerateOption) (generationConfig, error) {
	config := generationConfig{workers: min(4, runtime.GOMAXPROCS(0))}
	for _, option := range options {
		if option == nil {
			return generationConfig{}, fmt.Errorf("render workers option must not be nil")
		}
		if err := option(&config); err != nil {
			return generationConfig{}, err
		}
	}
	return config, nil
}
