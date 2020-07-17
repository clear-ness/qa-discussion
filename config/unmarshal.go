package config

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/clear-ness/qa-discussion/model"
)

func marshalConfig(cfg *model.Config) ([]byte, error) {
	return json.MarshalIndent(cfg, "", "    ")
}

func unmarshalConfig(r io.Reader) (*model.Config, error) {
	// Pre-flight check the syntax of the configuration file to improve error messaging.
	configData, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var config model.Config
	if err = json.Unmarshal(configData, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
