package app

import (
	"github.com/pkg/errors"

	"github.com/clear-ness/qa-discussion/store"
)

type Option func(s *Server) error

func StoreOverride(override interface{}) Option {
	return func(s *Server) error {
		switch o := override.(type) {
		case store.Store:
			s.newSqlStore = func() store.Store {
				return o
			}
			return nil

		case func(*Server) store.Store:
			s.newSqlStore = func() store.Store {
				return o(s)
			}
			return nil

		default:
			return errors.New("invalid StoreOverride")
		}
	}
}

type AppOption func(a *App)
type AppOptionCreator func() []AppOption

func ServerConnector(s *Server) AppOption {
	return func(a *App) {
		a.Srv = s
		a.Log = s.Log
		a.Cluster = s.Cluster
		a.HttpService = s.HTTPService
	}
}

func InitEmailBatching(interval *string) Option {
	return func(s *Server) error {
		if s.EmailBatching == nil {
			s.EmailBatching = NewEmailBatchingJob(s, interval)
		}

		return nil
	}
}
