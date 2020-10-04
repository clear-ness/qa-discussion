package app

import (
	"net/http"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/utils"
)

type TokenLocation int

const (
	TokenLocationNotFound TokenLocation = iota
	TokenLocationHeader
	TokenLocationCookie
	TokenLocationQueryString
)

func (a *App) IsPasswordValid(password string) *model.AppError {
	return utils.IsPasswordValidWithSettings(password, &a.Config().PasswordSettings)
}

func (a *App) authenticateUser(user *model.User, password string) (*model.User, *model.AppError) {
	if err := a.CheckPasswordAndAllCriteria(user, password); err != nil {
		err.StatusCode = http.StatusUnauthorized
		return user, err
	}

	return user, nil
}

func (a *App) CheckPasswordAndAllCriteria(user *model.User, password string) *model.AppError {
	if err := a.CheckUserPreflightAuthenticationCriteria(user); err != nil {
		return err
	}

	if err := a.checkUserPassword(user, password); err != nil {
		if passErr := a.Srv.Store.User().UpdateFailedPasswordAttempts(user.Id, user.FailedAttempts+1); passErr != nil {
			return passErr
		}

		a.ClearSessionCacheForUser(user.Id)

		return err
	}

	if passErr := a.Srv.Store.User().UpdateFailedPasswordAttempts(user.Id, 0); passErr != nil {
		return passErr
	}

	a.ClearSessionCacheForUser(user.Id)

	if err := a.CheckUserPostflightAuthenticationCriteria(user); err != nil {
		return err
	}

	return nil
}

func (a *App) CheckUserPreflightAuthenticationCriteria(user *model.User) *model.AppError {
	if err := checkUserNotDisabled(user); err != nil {
		return err
	}

	if err := checkUserLoginAttempts(user, *a.Config().ServiceSettings.MaximumLoginAttempts); err != nil {
		return err
	}

	return nil
}

func (a *App) CheckUserPostflightAuthenticationCriteria(user *model.User) *model.AppError {
	if !user.EmailVerified {
		return model.NewAppError("Login", "api.user.login.not_verified.app_error", nil, "user_id="+user.Id, http.StatusUnauthorized)
	}

	return nil
}

func checkUserLoginAttempts(user *model.User, max int) *model.AppError {
	if user.FailedAttempts >= max {
		return model.NewAppError("checkUserLoginAttempts", "api.user.check_user_login_attempts.too_many.app_error", nil, "user_id="+user.Id, http.StatusUnauthorized)
	}

	return nil
}

func checkUserNotDisabled(user *model.User) *model.AppError {
	if user.DeleteAt > 0 {
		return model.NewAppError("Login", "api.user.login.inactive.app_error", nil, "user_id="+user.Id, http.StatusUnauthorized)
	}

	return nil
}

func (a *App) checkUserPassword(user *model.User, password string) *model.AppError {
	if !model.ComparePassword(user.Password, password) {
		return model.NewAppError("checkUserPassword", "api.user.check_user_password.invalid.app_error", nil, "user_id="+user.Id, http.StatusUnauthorized)
	}

	return nil
}

func (a *App) DoubleCheckPassword(user *model.User, password string) *model.AppError {
	if err := checkUserLoginAttempts(user, *a.Config().ServiceSettings.MaximumLoginAttempts); err != nil {
		return err
	}

	if err := a.checkUserPassword(user, password); err != nil {
		if passErr := a.Srv.Store.User().UpdateFailedPasswordAttempts(user.Id, user.FailedAttempts+1); passErr != nil {
			return passErr
		}

		a.ClearSessionCacheForUser(user.Id)

		return err
	}

	if passErr := a.Srv.Store.User().UpdateFailedPasswordAttempts(user.Id, 0); passErr != nil {
		return passErr
	}

	a.ClearSessionCacheForUser(user.Id)

	return nil
}

func ParseAuthTokenFromRequest(r *http.Request) (string, TokenLocation) {
	//authHeader := r.Header.Get(model.HEADER_AUTH)

	if cookie, err := r.Cookie(model.SESSION_COOKIE_TOKEN); err == nil {
		return cookie.Value, TokenLocationCookie
	}

	return "", TokenLocationNotFound
}
