package app

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rs/cors"

	"github.com/clear-ness/qa-discussion/config"
	"github.com/clear-ness/qa-discussion/mlog"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
	"github.com/clear-ness/qa-discussion/store/cachelayer"
	"github.com/clear-ness/qa-discussion/store/searchlayer"
	"github.com/clear-ness/qa-discussion/store/sqlstore"
	"github.com/clear-ness/qa-discussion/utils"
)

type Server struct {
	sqlStore store.Store
	Store    store.Store

	RootRouter *mux.Router
	Router     *mux.Router

	Server      *http.Server
	ListenAddr  *net.TCPAddr
	RateLimiter *RateLimiter

	didFinishListen chan struct{}

	goroutineCount      int32
	goroutineExitSignal chan struct{}

	newSqlStore func() store.Store
	newStore    func() store.Store

	configStore config.Store

	Log *mlog.Logger

	EmailBatching *EmailBatchingJob
}

func NewServer(options ...Option) (*Server, error) {
	rootRouter := mux.NewRouter()

	s := &Server{
		goroutineExitSignal: make(chan struct{}, 1),
		RootRouter:          rootRouter,
	}

	if s.configStore == nil {
		configStore, err := config.NewFileStore("config.json")
		if err != nil {
			return nil, err
		}

		s.configStore = configStore
	}

	for _, option := range options {
		// テストの場合はnewStoreがセットされる
		// バッチの場合はEmailBatchingJobがセットされる
		if err := option(s); err != nil {
			return nil, errors.Wrap(err, "failed to apply option")
		}
	}

	//if s.Log == nil {
	//    s.Log = mlog.NewLogger()
	//}
	//mlog.RedirectStdLog(s.Log)
	//mlog.InitGlobalLogger(s.Log)

	//if err := utils.TranslationsPreInit(); err != nil {
	//    return nil, errors.Wrapf(err, "unable to load translation files")
	//}

	if s.newSqlStore == nil {
		s.newSqlStore = func() store.Store {
			return sqlstore.NewSqlSupplier(s.Config().SqlSettings)
		}
	}
	s.sqlStore = s.newSqlStore()

	if s.newStore == nil {
		s.newStore = func() store.Store {
			newLayer := searchlayer.NewSearchLayer(
				cachelayer.NewCacheLayer(
					s.sqlStore,
					s.Config(),
				),
				s.Config(),
			)

			newLayer.SetupIndexes()

			return newLayer
		}
	}
	s.Store = s.newStore()

	subpath, err := utils.GetSubpathFromConfig(s.Config())
	if err != nil {
		return nil, err
	}
	s.Router = s.RootRouter.PathPrefix(subpath).Subrouter()

	if s.EmailBatching != nil {
		if err := s.EmailBatching.scheduleJobs(); err != nil {
			return nil, err
		}
		s.EmailBatching.startJobs()
	}

	s.FakeApp().InitMigrations()

	return s, nil
}

var corsAllowedMethods = []string{
	"POST",
	"GET",
	"OPTIONS",
	"PUT",
	"PATCH",
	"DELETE",
}

func (s *Server) Start() error {
	var handler http.Handler = s.RootRouter

	if allowedOrigins := *s.Config().ServiceSettings.AllowCorsFrom; allowedOrigins != "" {
		exposedCorsHeaders := *s.Config().ServiceSettings.CorsExposedHeaders
		allowCredentials := *s.Config().ServiceSettings.CorsAllowCredentials
		debug := *s.Config().ServiceSettings.CorsDebug
		daySeconds := 86400
		corsWrapper := cors.New(cors.Options{
			AllowedOrigins:   strings.Fields(allowedOrigins),
			AllowedMethods:   corsAllowedMethods,
			AllowedHeaders:   []string{"*"},
			ExposedHeaders:   strings.Fields(exposedCorsHeaders),
			MaxAge:           daySeconds,
			AllowCredentials: allowCredentials,
			Debug:            debug,
		})

		// when debugging of CORS turned on then forward messages to logs
		if debug {
			//corsWrapper.Log = s.Log.StdLog(mlog.String("source", "cors"))
		}

		handler = corsWrapper.Handler(handler)
	}

	if *s.Config().RateLimitSettings.Enable {
		mlog.Info("RateLimiter is enabled")

		rateLimiter, err := NewRateLimiter(&s.Config().RateLimitSettings, s.Config().ServiceSettings.TrustedProxyIPHeader)
		if err != nil {
			return err
		}

		s.RateLimiter = rateLimiter
		handler = rateLimiter.RateLimitHandler(handler)
	}

	s.Server = &http.Server{
		Handler:      handler,
		ReadTimeout:  time.Duration(*s.Config().ServiceSettings.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(*s.Config().ServiceSettings.WriteTimeout) * time.Second,
	}

	addr := *s.Config().ServiceSettings.ListenAddress
	if addr == "" {
		if *s.Config().ServiceSettings.ConnectionSecurity == model.CONN_SECURITY_TLS {
			addr = ":https"
		} else {
			addr = ":http"
		}
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.ListenAddr = listener.Addr().(*net.TCPAddr)

	s.didFinishListen = make(chan struct{})
	go func() {
		err := s.Server.Serve(listener)

		if err != nil && err != http.ErrServerClosed {
			mlog.Critical("Error starting server", mlog.Err(err))
			time.Sleep(time.Second)
		}

		close(s.didFinishListen)
	}()

	return nil
}

func (s *Server) AppOptions() []AppOption {
	return []AppOption{
		ServerConnector(s),
	}
}

const TIME_TO_WAIT_FOR_CONNECTIONS_TO_CLOSE_ON_SERVER_SHUTDOWN = time.Second

func (s *Server) StopHTTPServer() {
	if s.Server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), TIME_TO_WAIT_FOR_CONNECTIONS_TO_CLOSE_ON_SERVER_SHUTDOWN)
		defer cancel()
		didShutdown := false
		for s.didFinishListen != nil && !didShutdown {
			if err := s.Server.Shutdown(ctx); err != nil {
				mlog.Warn("Unable to shutdown server", mlog.Err(err))
			}
			timer := time.NewTimer(time.Millisecond * 50)
			select {
			case <-s.didFinishListen:
				didShutdown = true
			case <-timer.C:
			}
			timer.Stop()
		}
		s.Server.Close()
		s.Server = nil
	}
}

func (s *Server) Shutdown() error {
	mlog.Info("Stopping Server...")

	s.StopHTTPServer()

	if s.EmailBatching != nil {
		s.EmailBatching.StopJobs()
	}

	s.WaitForGoroutines()

	if s.Store != nil {
		s.Store.Close()
	}

	mlog.Info("Server stopped")
	return nil
}

func (s *Server) Go(f func()) {
	atomic.AddInt32(&s.goroutineCount, 1)

	go func() {
		f()

		atomic.AddInt32(&s.goroutineCount, -1)
		select {
		case s.goroutineExitSignal <- struct{}{}:
		default:
		}
	}()
}

func (s *Server) WaitForGoroutines() {
	for atomic.LoadInt32(&s.goroutineCount) != 0 {
		<-s.goroutineExitSignal
	}
}

func (s *Server) FakeApp() *App {
	a := New(
		ServerConnector(s),
	)

	return a
}
