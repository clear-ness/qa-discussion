package app

import (
	"github.com/clear-ness/qa-discussion/clusters"
	"github.com/clear-ness/qa-discussion/mlog"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/services/httpservice"
)

type App struct {
	Srv *Server

	Log *mlog.Logger

	HttpService httpservice.HTTPService

	Cluster clusters.ClusterInterface

	Session   model.Session
	RequestId string
	// requested client's ip address
	IpAddress string
	// requested url
	Path           string
	UserAgent      string
	AcceptLanguage string
}

func New(options ...AppOption) *App {
	app := &App{}

	for _, option := range options {
		option(app)
	}

	return app
}

func (a *App) Shutdown() {
	a.Srv.Shutdown()
	a.Srv = nil
}
