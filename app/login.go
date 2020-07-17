package app

import (
	"fmt"
	"net/http"
	"time"

	"github.com/avct/uasurfer"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/utils"
)

func (a *App) GetUserForLogin(loginId string) (*model.User, *model.AppError) {

	if user, err := a.Srv.Store.User().GetForLogin(loginId); err == nil {
		return user, nil
	}

	return nil, model.NewAppError("GetUserForLogin", "store.sql_user.get_for_login.app_error", nil, "", http.StatusBadRequest)
}

func (a *App) AuthenticateUserForLogin(loginId, password string) (user *model.User, err *model.AppError) {
	if len(password) == 0 {
		return nil, model.NewAppError("AuthenticateUserForLogin", "api.user.login.blank_pwd.app_error", nil, "", http.StatusBadRequest)
	}

	if user, err = a.GetUserForLogin(loginId); err != nil {
		return nil, err
	}

	if user, err = a.authenticateUser(user, password); err != nil {
		return nil, err
	}

	return user, nil
}

func (a *App) DoLogin(w http.ResponseWriter, r *http.Request, user *model.User) *model.AppError {
	session := &model.Session{UserId: user.Id}
	session.GenerateCSRF()

	session.SetExpireInDays(*a.Config().ServiceSettings.SessionLengthWebInDays)

	ua := uasurfer.Parse(r.UserAgent())

	plat := getPlatformName(ua)
	os := getOSName(ua)
	bname := getBrowserName(ua, r.UserAgent())
	bversion := getBrowserVersion(ua, r.UserAgent())

	session.AddProp(model.SESSION_PROP_PLATFORM, plat)
	session.AddProp(model.SESSION_PROP_OS, os)
	session.AddProp(model.SESSION_PROP_BROWSER, fmt.Sprintf("%v/%v", bname, bversion))

	var err *model.AppError
	if session, err = a.CreateSession(session); err != nil {
		err.StatusCode = http.StatusInternalServerError
		return err
	}

	w.Header().Set(model.HEADER_TOKEN, session.Token)

	a.Session = *session

	return nil
}

func (a *App) AttachSessionCookies(w http.ResponseWriter, r *http.Request) {
	secure := false
	if GetProtocol(r) == "https" {
		secure = true
	}

	maxAge := *a.Config().ServiceSettings.SessionLengthWebInDays * 60 * 60 * 24
	domain := a.GetCookieDomain()
	subpath, _ := utils.GetSubpathFromConfig(a.Config())

	expiresAt := time.Unix(model.GetMillis()/1000+int64(maxAge), 0)
	sessionCookie := &http.Cookie{
		Name:     model.SESSION_COOKIE_TOKEN,
		Value:    a.Session.Token,
		Path:     subpath,
		MaxAge:   maxAge,
		Expires:  expiresAt,
		HttpOnly: true,
		Domain:   domain,
		Secure:   secure,
	}

	userCookie := &http.Cookie{
		Name:    model.SESSION_COOKIE_USER,
		Value:   a.Session.UserId,
		Path:    subpath,
		MaxAge:  maxAge,
		Expires: expiresAt,
		Domain:  domain,
		Secure:  secure,
	}

	csrfCookie := &http.Cookie{
		Name:    model.SESSION_COOKIE_CSRF,
		Value:   a.Session.GetCSRF(),
		Path:    subpath,
		MaxAge:  maxAge,
		Expires: expiresAt,
		Domain:  domain,
		Secure:  secure,
	}

	http.SetCookie(w, sessionCookie)
	http.SetCookie(w, userCookie)
	http.SetCookie(w, csrfCookie)
}

func GetProtocol(r *http.Request) string {
	if r.Header.Get(model.HEADER_FORWARDED_PROTO) == "https" || r.TLS != nil {
		return "https"
	}
	return "http"
}
