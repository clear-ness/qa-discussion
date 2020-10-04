package app

import (
	"context"
	"hash/maphash"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rs/cors"

	"github.com/clear-ness/qa-discussion/clusters"
	"github.com/clear-ness/qa-discussion/config"
	"github.com/clear-ness/qa-discussion/mlog"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/services/httpservice"
	"github.com/clear-ness/qa-discussion/services/l1cache"
	"github.com/clear-ness/qa-discussion/store"
	"github.com/clear-ness/qa-discussion/store/cachelayer"
	"github.com/clear-ness/qa-discussion/store/searchlayer"
	"github.com/clear-ness/qa-discussion/store/sqlstore"
	"github.com/clear-ness/qa-discussion/utils"
)

type Server struct {
	sqlStore store.Store
	Store    store.Store

	RootRouter      *mux.Router
	Router          *mux.Router
	WebSocketRouter *WebSocketRouter

	Server      *http.Server
	ListenAddr  *net.TCPAddr
	RateLimiter *RateLimiter

	didFinishListen chan struct{}

	goroutineCount      int32
	goroutineExitSignal chan struct{}

	newSqlStore func() store.Store
	newStore    func() store.Store

	sessionCache l1cache.Cache

	CacheProvider l1cache.Provider

	configStore config.Store

	Log *mlog.Logger

	EmailBatching *EmailBatchingJob

	HTTPService httpservice.HTTPService

	hubs     []*Hub
	hashSeed maphash.Seed

	Cluster   clusters.ClusterInterface
	clusterId string
}

func NewServer(options ...Option) (*Server, error) {
	rootRouter := mux.NewRouter()

	s := &Server{
		goroutineExitSignal: make(chan struct{}, 1),
		RootRouter:          rootRouter,
		hashSeed:            maphash.MakeSeed(),
		clusterId:           model.NewId(),
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

	s.CacheProvider = l1cache.NewProvider()

	s.sessionCache = s.CacheProvider.NewCache(&l1cache.CacheOptions{
		Size: model.SESSION_CACHE_SIZE,
	})

	if s.newSqlStore == nil {
		s.newSqlStore = func() store.Store {
			return sqlstore.NewSqlSupplier(s.Config().SqlSettings)
		}
	}
	s.sqlStore = s.newSqlStore()

	if s.newStore == nil {
		s.newStore = func() store.Store {
			newLayer := searchlayer.NewSearchLayer(
				// TODO: L1キャッシュ層を追加
				// L1 → L2(redisクラスタ) → DBの順にフォールバックさせる
				// https://nickcraver.com/blog/2016/02/17/stack-overflow-the-architecture-2016-edition/
				// We have an L1/L2 cache system with Redis. “L1” is HTTP Cache on the web servers or whatever application is in play.
				// “L2” is falling back to Redis and fetching the value out.
				// When one web server gets a cache miss in both L1 and L2, it fetches the value from source (a database query, API call, etc.)
				// and puts the result in both local cache and Redis. The next server wanting the value may miss L1, but would find the value in L2/Redis, saving a database query or API call.
				// We use redis's pub/sub to clear L1 caches on other servers when one web server does a removal for consistency,
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

	s.HTTPService = httpservice.MakeHTTPService(s)

	s.Cluster = clusters.MakeCluster(s)

	subpath, err := utils.GetSubpathFromConfig(s.Config())
	if err != nil {
		return nil, err
	}
	s.Router = s.RootRouter.PathPrefix(subpath).Subrouter()

	fakeApp := New(ServerConnector(s))
	fakeApp.HubStart()

	s.WebSocketRouter = &WebSocketRouter{
		server:   s,
		handlers: make(map[string]webSocketHandler),
	}
	s.WebSocketRouter.app = fakeApp

	if s.EmailBatching != nil {
		if err := s.EmailBatching.scheduleJobs(); err != nil {
			return nil, err
		}
		s.EmailBatching.startJobs()
	}

	s.FakeApp().InitMigrations()

	s.FakeApp().registerAllClusterMessageHandlers()
	// redis pub/subにsubscribeしておく
	s.Cluster.Start(s.clusterId)

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

	s.HubStop()

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

func (a *App) OriginChecker() func(*http.Request) bool {
	if allowed := *a.Config().ServiceSettings.AllowCorsFrom; allowed != "" {
		if allowed != "*" {
			siteURL, err := url.Parse(*a.Config().ServiceSettings.SiteURL)
			if err == nil {
				siteURL.Path = ""
				allowed += " " + siteURL.String()
			}
		}

		return utils.OriginChecker(allowed)
	}

	return nil
}
