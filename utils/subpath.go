package utils

import (
	"errors"
	"net/url"
	"path"

	"github.com/clear-ness/qa-discussion/model"
)

func GetSubpathFromConfig(config *model.Config) (string, error) {
	if config == nil {
		return "", errors.New("no config provided")
	} else if config.ServiceSettings.SiteURL == nil {
		return "/", nil
	}

	u, err := url.Parse(*config.ServiceSettings.SiteURL)
	if err != nil {
		return "", err
	}

	if u.Path == "" {
		return "/", nil
	}

	return path.Clean(u.Path), nil
}
