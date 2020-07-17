package web

import (
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"strings"

	"github.com/clear-ness/qa-discussion/app"
	"github.com/clear-ness/qa-discussion/mlog"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/utils"
)

func GetHandlerName(h func(*Context, http.ResponseWriter, *http.Request)) string {
	handlerName := runtime.FuncForPC(reflect.ValueOf(h).Pointer()).Name()
	pos := strings.LastIndex(handlerName, ".")
	if pos != -1 && len(handlerName) > pos {
		handlerName = handlerName[pos+1:]
	}
	return handlerName
}

type Handler struct {
	GetGlobalAppOptions app.AppOptionCreator
	HandleFunc          func(*Context, http.ResponseWriter, *http.Request)
	HandlerName         string
	RequireSession      bool
	TrustRequester      bool
	IsStatic            bool
	DisableWhenBusy     bool

	cspShaDirective string
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestID := model.NewId()
	mlog.Debug("Received HTTP request", mlog.String("method", r.Method), mlog.String("url", r.URL.Path), mlog.String("request_id", requestID))

	c := &Context{}
	c.App = app.New(
		h.GetGlobalAppOptions()...,
	)
	c.App.RequestId = requestID

	c.App.IpAddress = utils.GetIpAddress(r, c.App.Config().ServiceSettings.TrustedProxyIPHeader)

	c.App.UserAgent = r.UserAgent()
	c.App.AcceptLanguage = r.Header.Get("Accept-Language")
	c.Params = ParamsFromRequest(r)
	c.App.Path = r.URL.Path
	c.Log = c.App.Log

	subpath, _ := utils.GetSubpathFromConfig(c.App.Config())
	siteURLHeader := app.GetProtocol(r) + "://" + r.Host + subpath
	c.SetSiteURLHeader(siteURLHeader)

	w.Header().Set(model.HEADER_REQUEST_ID, c.App.RequestId)
	w.Header().Set(model.HEADER_VERSION_ID, fmt.Sprintf("%v", model.CurrentVersion))

	if !h.IsStatic {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "GET" {
			w.Header().Set("Expires", "0")
		}
	}

	token, tokenLocation := app.ParseAuthTokenFromRequest(r)

	if len(token) != 0 {
		session, err := c.App.GetSession(token)
		if err != nil {
			c.Log.Info("Invalid session", mlog.Err(err))
			if err.StatusCode == http.StatusInternalServerError {
				c.Err = err
			} else if h.RequireSession {
				c.RemoveSessionCookie(w, r)
				c.Err = model.NewAppError("ServeHTTP", "api.context.session_expired.app_error", nil, "token="+token, http.StatusUnauthorized)
			}
		} else {
			c.App.Session = *session
		}

		// rate limit also by userId
		if c.App.Srv.RateLimiter != nil && c.App.Srv.RateLimiter.UserIdRateLimit(c.App.Session.UserId, w) {
			return
		}

		h.checkCSRFToken(c, r, token, tokenLocation, session)
	}

	//c.Log = c.App.Log.With(
	//    mlog.String("path", c.App.Path),
	//    mlog.String("request_id", c.App.RequestId),
	//    mlog.String("ip_addr", c.App.IpAddress),
	//    mlog.String("user_id", c.App.Session.UserId),
	//    mlog.String("method", r.Method),
	//)

	if c.Err == nil && h.RequireSession {
		c.SessionRequired()
	}

	if c.Err == nil {
		h.HandleFunc(c, w, r)
	}

	if c.Err != nil {
		c.Err.RequestId = c.App.RequestId
		c.Err.Where = r.URL.Path

		if !*c.App.Config().ServiceSettings.EnableDeveloper {
			// hide internal error details
			c.Err.DetailedError = ""

			c.Err.Where = ""
		}

		if IsApiCall(c.App, r) {
			w.WriteHeader(c.Err.StatusCode)
			w.Write([]byte(c.Err.ToJson()))
		}
	}
}

func (h *Handler) checkCSRFToken(c *Context, r *http.Request, token string, tokenLocation app.TokenLocation, session *model.Session) (checked bool, passed bool) {
	csrfCheckNeeded := session != nil && c.Err == nil && tokenLocation == app.TokenLocationCookie && !h.TrustRequester && r.Method != "GET"
	csrfCheckPassed := false

	if csrfCheckNeeded {
		csrfHeader := r.Header.Get(model.HEADER_CSRF_TOKEN)

		// TODO: more secure
		if csrfHeader == session.GetCSRF() || r.Header.Get(model.HEADER_REQUESTED_WITH) == model.HEADER_REQUESTED_WITH_XML {
			csrfCheckPassed = true
		}

		if !csrfCheckPassed {
			c.App.Session = model.Session{}
			c.Err = model.NewAppError("ServeHTTP", "api.context.session_expired.app_error", nil, "token="+token+" Appears to be a CSRF attempt", http.StatusUnauthorized)
		}
	}

	return csrfCheckNeeded, csrfCheckPassed
}
