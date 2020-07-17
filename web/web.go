package web

import (
	"net/http"
	"path"
	"strings"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/services/configservice"
	"github.com/clear-ness/qa-discussion/utils"
	"github.com/gorilla/mux"
)

type Web struct {
	ConfigService configservice.ConfigService
	MainRouter    *mux.Router
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
