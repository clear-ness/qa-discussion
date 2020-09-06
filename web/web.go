package web

import (
	"net/http"
	"path"
	"strings"

	"github.com/clear-ness/qa-discussion/app"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/services/configservice"
	"github.com/clear-ness/qa-discussion/utils"
	"github.com/gorilla/mux"
)

type Web struct {
	GetGlobalAppOptions app.AppOptionCreator
	ConfigService       configservice.ConfigService
	MainRouter          *mux.Router
}

func New(config configservice.ConfigService, globalOptions app.AppOptionCreator, root *mux.Router) *Web {
	web := &Web{
		GetGlobalAppOptions: globalOptions,
		ConfigService:       config,
		MainRouter:          root,
	}

	web.InitOAuth()

	return web
}

func ReturnStatusOK(w http.ResponseWriter) {
	m := make(map[string]string)
	m[model.STATUS] = model.STATUS_OK
	w.Write([]byte(model.MapToJson(m)))
}

func IsApiCall(config configservice.ConfigService, r *http.Request) bool {
	subpath, _ := utils.GetSubpathFromConfig(config.Config())

	return strings.HasPrefix(r.URL.Path, path.Join(subpath, "api")+"/")
}

func IsOAuthApiCall(config configservice.ConfigService, r *http.Request) bool {
	subpath, _ := utils.GetSubpathFromConfig(config.Config())

	if r.Method == "POST" && r.URL.Path == path.Join(subpath, "oauth", "authorize") {
		return true
	}
	if r.URL.Path == path.Join(subpath, "oauth", "apps", "authorized") ||
		r.URL.Path == path.Join(subpath, "oauth", "deauthorize") ||
		r.URL.Path == path.Join(subpath, "oauth", "access_token") {
		return true
	}

	return false
}
