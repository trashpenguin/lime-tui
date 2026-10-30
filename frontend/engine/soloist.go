package engine

import (
	"context"
	"sync"
	"time"
)

// SoloistConfig is private local configuration. Never log or include it in reports.
type SoloistConfig struct {
	Path string `json:"path"`
	Key  string `json:"key"`
}

var soloistDefaults struct {
	sync.RWMutex
	config SoloistConfig
	err    error
}

func ConfigureSoloist(config SoloistConfig, err error) {
	soloistDefaults.Lock()
	defer soloistDefaults.Unlock()
	soloistDefaults.config, soloistDefaults.err = config, err
}

func currentSoloistConfig() (SoloistConfig, error) {
	soloistDefaults.RLock()
	defer soloistDefaults.RUnlock()
	return soloistDefaults.config, soloistDefaults.err
}

func (e *Engine) Soloist() bool { return e.soloist.Path != "" }

func (e *Engine) startSoloist() error {
	return e.startSoloistContext(context.Background())
}

func (e *Engine) startSoloistContext(parent context.Context) error {
	ctx, cancel := context.WithTimeout(parent, 40*time.Second)
	defer cancel()
	return e.rpc.call(ctx, "soloist_start", e.soloist, nil)
}
