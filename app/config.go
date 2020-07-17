package app

import (
	"github.com/clear-ness/qa-discussion/model"
)

func (a *App) Config() *model.Config {
	return a.Srv.Config()
}

func (s *Server) Config() *model.Config {
	return s.configStore.Get()
}

func (a *App) GetSiteURL() string {
	return *a.Config().ServiceSettings.SiteURL
}

func (a *App) GetCookieDomain() string {
	return ""
}
