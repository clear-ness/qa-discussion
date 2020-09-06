package web

import (
	"net/http"

	"github.com/clear-ness/qa-discussion/model"
)

func (w *Web) InitOAuth() {
	// リクエストbodyからAuthorizeRequestを作成し、authDataを作成する
	w.MainRouter.Handle("/oauth/authorize", w.ApiSessionRequired(authorizeOAuthApp)).Methods("POST")

	// users can revoke authorization from their setting.
	w.MainRouter.Handle("/oauth/deauthorize", w.ApiSessionRequired(deauthorizeOAuthApp)).Methods("POST")

	// accessDataをDB保存し、
	// OAuth consumerサーバーがaccess tokenを取得する (既にexpired access tokenだったとしても)
	w.MainRouter.Handle("/oauth/access_token", w.ApiHandlerTrustRequester(getAccessToken)).Methods("POST")
}

func authorizeOAuthApp(c *Context, w http.ResponseWriter, r *http.Request) {
	authRequest := model.AuthorizeRequestFromJson(r.Body)
	if authRequest == nil {
		c.SetInvalidParam("authorize_request")
	}

	if err := authRequest.IsValid(); err != nil {
		c.Err = err
		return
	}

	if c.App.Session.IsOAuth {
		c.SetPermissionError(model.PERMISSION_EDIT_OTHER_USERS)
		c.Err.DetailedError += ", attempted access by oauth app"
		return
	}

	// auth code grantフローの場合は
	// authDataをDB保存し、authData.CodeやauthData.Stateを含めたredirect uriを取得
	// (後にaccessDataを作成)
	redirectUrl, err := c.App.AllowOAuthAppAccessToUser(c.App.Session.UserId, authRequest)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(model.MapToJson(map[string]string{"redirect": redirectUrl})))
}

func getAccessToken(c *Context, w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	code := r.FormValue("code")
	refreshToken := r.FormValue("refresh_token")

	grantType := r.FormValue("grant_type")
	switch grantType {
	case model.ACCESS_TOKEN_GRANT_TYPE:
		if len(code) == 0 {
			c.Err = model.NewAppError("getAccessToken", "api.oauth.get_access_token.missing_code.app_error", nil, "", http.StatusBadRequest)
			return
		}
	case model.REFRESH_TOKEN_GRANT_TYPE:
		if len(refreshToken) == 0 {
			c.Err = model.NewAppError("getAccessToken", "api.oauth.get_access_token.missing_refresh_token.app_error", nil, "", http.StatusBadRequest)
			return
		}
	default:
		c.Err = model.NewAppError("getAccessToken", "api.oauth.get_access_token.bad_grant.app_error", nil, "", http.StatusBadRequest)
		return
	}

	clientId := r.FormValue("client_id")
	if len(clientId) != 26 {
		c.Err = model.NewAppError("getAccessToken", "api.oauth.get_access_token.bad_client_id.app_error", nil, "", http.StatusBadRequest)
		return
	}

	secret := r.FormValue("client_secret")
	if len(secret) == 0 {
		c.Err = model.NewAppError("getAccessToken", "api.oauth.get_access_token.bad_client_secret.app_error", nil, "", http.StatusBadRequest)
		return
	}

	redirectUri := r.FormValue("redirect_uri")

	// codeからauthDataをDB取得し、
	// そのauthDataを元にしてaccessDataをDB作成し、
	// 不要になったauthDataをDB削除し、
	// AccessResponseを取得する。
	accessRsp, err := c.App.GetOAuthAccessTokenForCodeFlow(clientId, grantType, redirectUri, code, secret, refreshToken)
	if err != nil {
		c.Err = err
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	w.Write([]byte(accessRsp.ToJson()))
}

func deauthorizeOAuthApp(c *Context, w http.ResponseWriter, r *http.Request) {
	requestData := model.MapFromJson(r.Body)
	clientId := requestData["client_id"]

	if len(clientId) != 26 {
		c.SetInvalidParam("client_id")
		return
	}

	err := c.App.DeauthorizeOAuthAppForUser(c.App.Session.UserId, clientId)
	if err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}
