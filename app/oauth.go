package app

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/clear-ness/qa-discussion/model"
)

func (a *App) CreateOAuthApp(app *model.OAuthApp) (*model.OAuthApp, *model.AppError) {
	app.ClientSecret = model.NewId()

	return a.Srv.Store.OAuth().SaveApp(app)
}

func (a *App) GetOAuthApp(appId string) (*model.OAuthApp, *model.AppError) {
	return a.Srv.Store.OAuth().GetApp(appId)
}

func (a *App) UpdateOauthApp(oldApp, updatedApp *model.OAuthApp) (*model.OAuthApp, *model.AppError) {
	updatedApp.Id = oldApp.Id
	updatedApp.UserId = oldApp.UserId
	updatedApp.CreateAt = oldApp.CreateAt
	updatedApp.ClientSecret = oldApp.ClientSecret

	return a.Srv.Store.OAuth().UpdateApp(updatedApp)
}

func (a *App) GetOAuthAppsByUserId(userId string, page, perPage int) ([]*model.OAuthApp, *model.AppError) {
	return a.Srv.Store.OAuth().GetAppByUserId(userId, page*perPage, perPage)
}

func (a *App) DeleteOAuthApp(appId string) *model.AppError {
	if err := a.Srv.Store.OAuth().DeleteApp(appId); err != nil {
		return err
	}

	return nil
}

func (a *App) RegenerateOAuthAppSecret(app *model.OAuthApp) (*model.OAuthApp, *model.AppError) {
	app.ClientSecret = model.NewId()
	if _, err := a.Srv.Store.OAuth().UpdateApp(app); err != nil {
		return nil, err
	}

	return app, nil
}

func (a *App) GetAuthorizedAppsForUser(userId string, page, perPage int) ([]*model.OAuthApp, *model.AppError) {
	apps, err := a.Srv.Store.OAuth().GetAuthorizedApps(userId, page*perPage, perPage)
	if err != nil {
		return nil, err
	}

	for k, a := range apps {
		a.Sanitize()
		apps[k] = a
	}

	return apps, nil
}

func (a *App) RevokeAccessToken(token string) *model.AppError {
	schan := make(chan *model.AppError, 1)
	go func() {
		schan <- a.Srv.Store.Session().Remove(token)
		close(schan)
	}()

	if _, err := a.Srv.Store.OAuth().GetAccessData(token); err != nil {
		return model.NewAppError("RevokeAccessToken", "api.oauth.revoke_access_token.get.app_error", nil, "", http.StatusBadRequest)
	}

	if err := a.Srv.Store.OAuth().RemoveAccessData(token); err != nil {
		return model.NewAppError("RevokeAccessToken", "api.oauth.revoke_access_token.del_token.app_error", nil, "", http.StatusInternalServerError)
	}

	if err := <-schan; err != nil {
		return model.NewAppError("RevokeAccessToken", "api.oauth.revoke_access_token.del_session.app_error", nil, "", http.StatusInternalServerError)
	}

	return nil
}

func (a *App) AllowOAuthAppAccessToUser(userId string, authRequest *model.AuthorizeRequest) (string, *model.AppError) {
	if len(authRequest.Scope) == 0 {
		authRequest.Scope = model.DEFAULT_SCOPE
	}

	oauthApp, err := a.Srv.Store.OAuth().GetApp(authRequest.ClientId)
	if err != nil {
		return "", err
	}

	if !oauthApp.IsValidRedirectURL(authRequest.RedirectUri) {
		return "", model.NewAppError("AllowOAuthAppAccessToUser", "api.oauth.allow_oauth.redirect_callback.app_error", nil, "", http.StatusBadRequest)
	}

	var redirectURI string
	switch authRequest.ResponseType {
	case model.AUTHCODE_RESPONSE_TYPE:
		// authDataをDB保存し、authData.CodeやauthData.Stateを含めたredirect uriを取得。
		redirectURI, err = a.GetOAuthCodeRedirect(userId, authRequest)
	case model.IMPLICIT_RESPONSE_TYPE:
		// implicitの場合はauthDataは扱わない。
		redirectURI, err = a.GetOAuthImplicitRedirect(userId, authRequest)
	default:
		return authRequest.RedirectUri + "?error=unsupported_response_type&state=" + authRequest.State, nil
	}
	if err != nil {
		return authRequest.RedirectUri + "?error=server_error&state=" + authRequest.State, nil
	}

	authorizedApp := &model.OAuthAuthorizedApp{
		UserId:   userId,
		ClientId: authRequest.ClientId,
		Scope:    authRequest.Scope,
	}

	if err = a.Srv.Store.OAuth().SaveAuthorizedApp(authorizedApp); err != nil {
		return authRequest.RedirectUri + "?error=server_error&state=" + authRequest.State, nil
	}

	return redirectURI, nil
}

func (a *App) GetOAuthCodeRedirect(userId string, authRequest *model.AuthorizeRequest) (string, *model.AppError) {
	authData := &model.AuthData{UserId: userId, ClientId: authRequest.ClientId, CreateAt: model.GetMillis(), RedirectUri: authRequest.RedirectUri, State: authRequest.State, Scope: authRequest.Scope}
	authData.Code = model.NewId() + model.NewId()

	if _, err := a.Srv.Store.OAuth().SaveAuthData(authData); err != nil {
		return authRequest.RedirectUri + "?error=server_error&state=" + authRequest.State, nil
	}

	return authRequest.RedirectUri + "?code=" + url.QueryEscape(authData.Code) + "&state=" + url.QueryEscape(authData.State), nil
}

func (a *App) GetOAuthImplicitRedirect(userId string, authRequest *model.AuthorizeRequest) (string, *model.AppError) {
	session, err := a.GetOAuthAccessTokenForImplicitFlow(userId, authRequest)
	if err != nil {
		return "", err
	}

	values := &url.Values{}
	values.Add("access_token", session.Token)
	values.Add("token_type", "bearer")
	values.Add("expires_in", strconv.FormatInt((session.ExpiresAt-model.GetMillis())/1000, 10))
	values.Add("scope", authRequest.Scope)
	values.Add("state", authRequest.State)

	return fmt.Sprintf("%s#%s", authRequest.RedirectUri, values.Encode()), nil
}

func (a *App) GetOAuthAccessTokenForImplicitFlow(userId string, authRequest *model.AuthorizeRequest) (*model.Session, *model.AppError) {
	oauthApp, err := a.GetOAuthApp(authRequest.ClientId)
	if err != nil {
		return nil, model.NewAppError("GetOAuthAccessToken", "api.oauth.get_access_token.credentials.app_error", nil, "", http.StatusNotFound)
	}

	user, err := a.GetUser(userId)
	if err != nil {
		return nil, err
	}

	session, err := a.newSession(oauthApp.Name, user)
	if err != nil {
		return nil, err
	}

	accessData := &model.AccessData{ClientId: authRequest.ClientId, UserId: user.Id, Token: session.Token, RefreshToken: "", RedirectUri: authRequest.RedirectUri, ExpiresAt: session.ExpiresAt, Scope: authRequest.Scope}

	if _, err := a.Srv.Store.OAuth().SaveAccessData(accessData); err != nil {
		return nil, model.NewAppError("GetOAuthAccessToken", "api.oauth.get_access_token.internal_saving.app_error", nil, "", http.StatusInternalServerError)
	}

	return session, nil
}

func (a *App) newSession(appName string, user *model.User) (*model.Session, *model.AppError) {
	session := &model.Session{UserId: user.Id, IsOAuth: true}
	session.GenerateCSRF()
	session.SetExpireInDays(*a.Config().ServiceSettings.SessionLengthOAuthInDays)

	session.AddProp(model.SESSION_PROP_PLATFORM, appName)
	session.AddProp(model.SESSION_PROP_OS, "OAuth2")
	session.AddProp(model.SESSION_PROP_BROWSER, "OAuth2")

	session, err := a.Srv.Store.Session().Save(session)
	if err != nil {
		return nil, model.NewAppError("newSession", "api.oauth.get_access_token.internal_session.app_error", nil, "", http.StatusInternalServerError)
	}

	return session, nil
}

func (a *App) GetOAuthAccessTokenForCodeFlow(clientId, grantType, redirectUri, code, secret, refreshToken string) (*model.AccessResponse, *model.AppError) {
	// clientId はOAuthApps.Idカラムに相当する。
	oauthApp, err := a.Srv.Store.OAuth().GetApp(clientId)
	if err != nil {
		return nil, model.NewAppError("GetOAuthAccessToken", "api.oauth.get_access_token.credentials.app_error", nil, "", http.StatusNotFound)
	}

	if oauthApp.ClientSecret != secret {
		return nil, model.NewAppError("GetOAuthAccessToken", "api.oauth.get_access_token.credentials.app_error", nil, "", http.StatusForbidden)
	}

	var user *model.User
	var accessData *model.AccessData
	var accessRsp *model.AccessResponse

	if grantType == model.ACCESS_TOKEN_GRANT_TYPE {
		var authData *model.AuthData
		authData, err = a.Srv.Store.OAuth().GetAuthData(code)
		if err != nil {
			return nil, model.NewAppError("GetOAuthAccessToken", "api.oauth.get_access_token.expired_code.app_error", nil, "", http.StatusBadRequest)
		}

		if authData.IsExpired() {
			a.Srv.Store.OAuth().RemoveAuthData(authData.Code)
			return nil, model.NewAppError("GetOAuthAccessToken", "api.oauth.get_access_token.expired_code.app_error", nil, "", http.StatusForbidden)
		}

		if authData.RedirectUri != redirectUri {
			return nil, model.NewAppError("GetOAuthAccessToken", "api.oauth.get_access_token.redirect_uri.app_error", nil, "", http.StatusBadRequest)
		}

		user, err = a.Srv.Store.User().Get(authData.UserId)
		if err != nil {
			return nil, model.NewAppError("GetOAuthAccessToken", "api.oauth.get_access_token.internal_user.app_error", nil, "", http.StatusNotFound)
		}

		accessData, err = a.Srv.Store.OAuth().GetPreviousAccessData(user.Id, clientId)
		if err != nil {
			return nil, model.NewAppError("GetOAuthAccessToken", "api.oauth.get_access_token.internal.app_error", nil, "", http.StatusBadRequest)
		}

		if accessData != nil {
			if accessData.IsExpired() {
				var access *model.AccessResponse

				access, err = a.newSessionUpdateToken(oauthApp.Name, accessData, user)
				if err != nil {
					return nil, err
				}

				accessRsp = access
			} else {
				accessRsp = &model.AccessResponse{
					AccessToken:  accessData.Token,
					TokenType:    model.ACCESS_TOKEN_TYPE,
					RefreshToken: accessData.RefreshToken,
					ExpiresIn:    int32((accessData.ExpiresAt - model.GetMillis()) / 1000),
				}
			}
		} else {
			var session *model.Session
			session, err = a.newSession(oauthApp.Name, user)
			if err != nil {
				return nil, err
			}

			// authData.ScopeをaccessData.Scopeとして使う。
			// session.TokenをaccessData.Tokenとして使う。
			accessData = &model.AccessData{ClientId: clientId, UserId: user.Id, Token: session.Token, RefreshToken: model.NewId(), RedirectUri: redirectUri, ExpiresAt: session.ExpiresAt, Scope: authData.Scope}

			if _, err = a.Srv.Store.OAuth().SaveAccessData(accessData); err != nil {
				return nil, model.NewAppError("GetOAuthAccessToken", "api.oauth.get_access_token.internal_saving.app_error", nil, "", http.StatusInternalServerError)
			}

			accessRsp = &model.AccessResponse{
				AccessToken:  session.Token,
				TokenType:    model.ACCESS_TOKEN_TYPE,
				RefreshToken: accessData.RefreshToken,
				ExpiresIn:    int32(*a.Config().ServiceSettings.SessionLengthOAuthInDays * 60 * 60 * 24),
			}
		}

		a.Srv.Store.OAuth().RemoveAuthData(authData.Code)
	} else {
		// when grantType is refresh_token
		accessData, err = a.Srv.Store.OAuth().GetAccessDataByRefreshToken(refreshToken)
		if err != nil {
			return nil, model.NewAppError("GetOAuthAccessToken", "api.oauth.get_access_token.refresh_token.app_error", nil, "", http.StatusNotFound)
		}

		user, err := a.Srv.Store.User().Get(accessData.UserId)
		if err != nil {
			return nil, model.NewAppError("GetOAuthAccessToken", "api.oauth.get_access_token.internal_user.app_error", nil, "", http.StatusNotFound)
		}

		access, err := a.newSessionUpdateToken(oauthApp.Name, accessData, user)
		if err != nil {
			return nil, err
		}

		accessRsp = access
	}

	return accessRsp, nil
}

func (a *App) newSessionUpdateToken(appName string, accessData *model.AccessData, user *model.User) (*model.AccessResponse, *model.AppError) {
	// remove the previous session
	a.Srv.Store.Session().Remove(accessData.Token)

	session, err := a.newSession(appName, user)
	if err != nil {
		return nil, err
	}
	accessData.Token = session.Token
	accessData.RefreshToken = model.NewId()
	accessData.ExpiresAt = session.ExpiresAt

	if _, err := a.Srv.Store.OAuth().UpdateAccessData(accessData); err != nil {
		return nil, model.NewAppError("newSessionUpdateToken", "web.get_access_token.internal_saving.app_error", nil, "", http.StatusInternalServerError)
	}

	accessRsp := &model.AccessResponse{
		AccessToken:  session.Token,
		RefreshToken: accessData.RefreshToken,
		TokenType:    model.ACCESS_TOKEN_TYPE,
		ExpiresIn:    int32(*a.Config().ServiceSettings.SessionLengthOAuthInDays * 60 * 60 * 24),
	}

	return accessRsp, nil
}

func (a *App) DeauthorizeOAuthAppForUser(userId, appId string) *model.AppError {
	// Revoke app sessions
	// accessDataは(clientId,userId)ペアがユニークキー。
	// ここでは0個または1個取得出来る。
	accessData, err := a.Srv.Store.OAuth().GetAccessDataByUserForApp(userId, appId)
	if err != nil {
		return err
	}

	for _, ad := range accessData {
		// accessDataを削除し、
		// accessData.Tokenにsession.Tokenが一致するsessionも削除。
		if err := a.RevokeAccessToken(ad.Token); err != nil {
			return err
		}

		// トークンの一致するaccessDataを削除
		if err := a.Srv.Store.OAuth().RemoveAccessData(ad.Token); err != nil {
			return err
		}
	}

	if err := a.Srv.Store.OAuth().DeleteAuthorizedApp(userId, appId); err != nil {
		return err
	}

	return nil
}
