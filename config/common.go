package config

import (
	"io"
	"sync"

	"github.com/clear-ness/qa-discussion/model"
)

type commonStore struct {
	configLock sync.RWMutex
	config     *model.Config
}

func (cs *commonStore) load(f io.ReadCloser, needsSave bool, validate func(*model.Config) error, persist func(*model.Config) error) error {
	loadedCfg, err := unmarshalConfig(f)
	if err != nil {
		return err
	}

	loadedCfg.SetDefaults()

	if validate != nil {
		if err = validate(loadedCfg); err != nil {
			return err
		}
	}

	cs.configLock.Lock()
	var unlockOnce sync.Once
	defer unlockOnce.Do(cs.configLock.Unlock)

	if needsSave && persist != nil {
		if err = persist(loadedCfg); err != nil {
			return err
		}
	}

	cs.config = loadedCfg
	unlockOnce.Do(cs.configLock.Unlock)

	return nil
}

func (cs *commonStore) Get() *model.Config {
	cs.configLock.RLock()
	defer cs.configLock.RUnlock()

	return cs.config
}

func (cs *commonStore) validate(cfg *model.Config) error {
	if err := cfg.IsValid(); err != nil {
		return err
	}

	return nil
}
