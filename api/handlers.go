package api

import (
	"net/http"

	"github.com/clear-ness/qa-discussion/web"
)

type Context = web.Context

// session not requires, and csrfCheckNeeded
func (api *API) ApiHandler(h func(*Context, http.ResponseWriter, *http.Request)) http.Handler {
	handler := &web.Handler{
		GetGlobalAppOptions: api.GetGlobalAppOptions,
		HandleFunc:          h,
		HandlerName:         web.GetHandlerName(h),
		RequireSession:      false,
		TrustRequester:      false,
		IsStatic:            false,
	}

	return handler
}

// session requires, and csrfCheckNeeded
func (api *API) ApiSessionRequired(h func(*Context, http.ResponseWriter, *http.Request)) http.Handler {
	handler := &web.Handler{
		GetGlobalAppOptions: api.GetGlobalAppOptions,
		HandleFunc:          h,
		HandlerName:         web.GetHandlerName(h),
		RequireSession:      true,
		TrustRequester:      false,
		IsStatic:            false,
	}

	return handler
}

// session requires, and not csrfCheckNeeded
func (api *API) ApiSessionRequiredTrustRequester(h func(*Context, http.ResponseWriter, *http.Request)) http.Handler {
	handler := &web.Handler{
		GetGlobalAppOptions: api.GetGlobalAppOptions,
		HandleFunc:          h,
		HandlerName:         web.GetHandlerName(h),
		RequireSession:      true,
		TrustRequester:      true,
		IsStatic:            false,
	}

	return handler
}
