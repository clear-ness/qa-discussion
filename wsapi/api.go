package wsapi

import (
	"github.com/clear-ness/qa-discussion/app"
)

type API struct {
	App    *app.App
	Router *app.WebSocketRouter
}

func Init(s *app.Server) {
	a := app.New(app.ServerConnector(s))

	api := &API{
		App:    a,
		Router: s.WebSocketRouter,
	}

	api.InitUser()
	api.InitSystem()
}
