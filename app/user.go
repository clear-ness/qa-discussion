package app

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/clear-ness/qa-discussion/mlog"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"

	"github.com/disintegration/imaging"
)

const (
	TOKEN_TYPE_VERIFY_EMAIL      = "verify_email"
	TOKEN_TYPE_PASSWORD_RECOVERY = "password_recovery"

	PASSWORD_RECOVER_EXPIRY_TIME = 1000 * 60 * 60 // 1 hour
)

func (a *App) CreateUserFromSignup(user *model.User) (*model.User, *model.AppError) {
	user.EmailVerified = false
	user.Points = 0

	ruser, err := a.CreateNormalUser(user)
	if err != nil {
		return nil, err
	}

	if err := a.SendWelcomeEmail(ruser.Id, ruser.Email, ruser.EmailVerified, a.GetSiteURL()); err != nil {
		mlog.Error("Failed to send welcome email on create user from signup", mlog.Err(err))
	}

	return ruser, nil
}

func (a *App) SendEmailVerification(user *model.User, newEmail string) *model.AppError {
	token, err := a.CreateVerifyEmailToken(user.Id, newEmail)
	if err != nil {
		return err
	}

	return a.SendVerifyEmail(newEmail, a.GetSiteURL(), token.Token)
}

func (a *App) CreateVerifyEmailToken(userId string, newEmail string) (*model.Token, *model.AppError) {
	tokenExtra := struct {
		UserId string
		Email  string
	}{
		userId,
		newEmail,
	}
	jsonData, err := json.Marshal(tokenExtra)

	if err != nil {
		return nil, model.NewAppError("CreateVerifyEmailToken", "api.user.create_email_token.error", nil, "", http.StatusInternalServerError)
	}

	token := model.NewToken(TOKEN_TYPE_VERIFY_EMAIL, string(jsonData))

	if err := a.Srv.Store.Token().Save(token); err != nil {
		return nil, err
	}

	return token, nil
}

func (a *App) GetVerifyEmailToken(token string) (*model.Token, *model.AppError) {
	rtoken, err := a.Srv.Store.Token().GetByToken(token)
	if err != nil {
		return nil, model.NewAppError("GetVerifyEmailToken", "api.user.verify_email.bad_link.app_error", nil, err.Error(), http.StatusBadRequest)
	}
	if rtoken.Type != TOKEN_TYPE_VERIFY_EMAIL {
		return nil, model.NewAppError("GetVerifyEmailToken", "api.user.verify_email.broken_token.app_error", nil, "", http.StatusBadRequest)
	}
	return rtoken, nil
}

func (a *App) VerifyEmailFromToken(userSuppliedTokenString string) *model.AppError {
	token, err := a.GetVerifyEmailToken(userSuppliedTokenString)
	if err != nil {
		return err
	}
	if model.GetMillis()-token.CreateAt >= PASSWORD_RECOVER_EXPIRY_TIME {
		return model.NewAppError("VerifyEmailFromToken", "api.user.verify_email.link_expired.app_error", nil, "", http.StatusBadRequest)
	}

	tokenData := struct {
		UserId string
		Email  string
	}{}

	err2 := json.Unmarshal([]byte(token.Extra), &tokenData)
	if err2 != nil {
		return model.NewAppError("VerifyEmailFromToken", "api.user.verify_email.token_parse.error", nil, "", http.StatusInternalServerError)
	}

	user, err := a.GetUser(tokenData.UserId)
	if err != nil {
		return err
	}

	if err := a.VerifyUserEmail(tokenData.UserId, tokenData.Email); err != nil {
		return err
	}

	if user.Email != tokenData.Email {
		a.Srv.Go(func() {
			if err := a.SendEmailChangeEmail(user.Email, tokenData.Email, a.GetSiteURL()); err != nil {
				mlog.Error("Failed to send email change email", mlog.Err(err))
			}
		})
	}

	if err := a.DeleteToken(token); err != nil {
		mlog.Error("Failed to delete token", mlog.Err(err))
	}

	return nil
}

func (a *App) GetUser(userId string) (*model.User, *model.AppError) {
	user, err := a.Srv.Store.User().Get(userId)
	user.ProfileImageLink = user.GetProfileImageLink(&a.Config().FileSettings)
	return user, err
}

func (a *App) GetUserByEmail(email string) (*model.User, *model.AppError) {
	user, err := a.Srv.Store.User().GetByEmail(email)
	if err != nil {
		if err.Id == store.MISSING_ACCOUNT_ERROR {
			err.StatusCode = http.StatusNotFound
			return nil, err
		}
		err.StatusCode = http.StatusBadRequest
		return nil, err
	}

	user.ProfileImageLink = user.GetProfileImageLink(&a.Config().FileSettings)

	return user, nil
}

func (a *App) GetUsers(userIds []string) ([]*model.User, *model.AppError) {
	users, err := a.Srv.Store.User().GetByIds(userIds)
	for _, user := range users {
		user.ProfileImageLink = user.GetProfileImageLink(&a.Config().FileSettings)
	}
	return users, err
}

func (a *App) GetUsersByDates(options *model.GetUsersOptions) ([]*model.User, *model.AppError) {
	users, err := a.Srv.Store.User().GetUsersByDates(options)
	for _, user := range users {
		user.ProfileImageLink = user.GetProfileImageLink(&a.Config().FileSettings)

		sanitizeOptions := map[string]bool{}
		sanitizeOptions["email"] = false
		user.SanitizeProfile(sanitizeOptions)
	}

	return users, err
}

func (a *App) VerifyUserEmail(userId, email string) *model.AppError {
	_, err := a.Srv.Store.User().VerifyEmail(userId, email)
	if err != nil {
		return err
	}

	return nil
}

func (a *App) GetPasswordRecoveryToken(token string) (*model.Token, *model.AppError) {
	rtoken, err := a.Srv.Store.Token().GetByToken(token)
	if err != nil {
		return nil, model.NewAppError("GetPasswordRecoveryToken", "api.user.reset_password.invalid_link.app_error", nil, err.Error(), http.StatusBadRequest)
	}

	if rtoken.Type != TOKEN_TYPE_PASSWORD_RECOVERY {
		return nil, model.NewAppError("GetPasswordRecoveryToken", "api.user.reset_password.invalid_token.app_error", nil, "", http.StatusBadRequest)
	}

	return rtoken, nil
}

func (a *App) DeleteToken(token *model.Token) *model.AppError {
	return a.Srv.Store.Token().Delete(token.Token)
}

func (a *App) CreateNormalUser(user *model.User) (*model.User, *model.AppError) {
	user.Type = model.USER_TYPE_NORMAL

	if *a.Config().ServiceSettings.EnableAdminUser {
		count, err := a.Srv.Store.User().Count(&model.UserCountOptions{IncludeDeleted: true})
		if err != nil {
			return nil, err
		}
		if count <= 0 {
			user.Type = model.USER_TYPE_ADMIN
		}
	}

	ruser, err := a.createUser(user)
	if err != nil {
		return nil, err
	}

	return ruser, nil
}

func (a *App) createUser(user *model.User) (*model.User, *model.AppError) {
	if err := a.IsPasswordValid(user.Password); err != nil {
		return nil, err
	}

	ruser, err := a.Srv.Store.User().Save(user)
	if err != nil {
		mlog.Error("Couldn't save the user", mlog.Err(err))
		return nil, err
	}

	ruser.Sanitize(map[string]bool{})
	return ruser, nil
}

func (a *App) UpdateUser(user *model.User) (*model.User, *model.AppError) {
	prev, err := a.GetUser(user.Id)
	if err != nil {
		return nil, err
	}

	newEmail := ""
	if prev.Email != user.Email {
		newEmail = user.Email

		_, err = a.GetUserByEmail(newEmail)
		if err == nil {
			return nil, model.NewAppError("UpdateUser", "store.sql_user.update.email_taken.app_error", nil, "user_id="+user.Id, http.StatusBadRequest)
		}
	}

	userUpdate, err := a.Srv.Store.User().Update(user, false)
	if err != nil {
		return nil, err
	}

	if userUpdate.New.Email != userUpdate.Old.Email || newEmail != "" {
		a.Srv.Go(func() {
			if err := a.SendEmailVerification(userUpdate.New, newEmail); err != nil {
				mlog.Error("Failed to send email verification", mlog.Err(err))
			}
		})
	}

	if userUpdate.New.Username != userUpdate.Old.Username {
		a.Srv.Go(func() {
			if err := a.SendChangeUsernameEmail(userUpdate.Old.Username, userUpdate.New.Username, userUpdate.New.Email, a.GetSiteURL()); err != nil {
				mlog.Error("Failed to send change username email", mlog.Err(err))
			}
		})
	}

	return userUpdate.New, nil
}

func (a *App) UpdateUserType(userId string, newType string) (*model.User, *model.AppError) {
	user, err := a.GetUser(userId)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, model.NewAppError("UpdateUserType", "api.user.update_user_type.get.app_error", nil, "", http.StatusNotFound)
	}

	if user.Type == model.USER_TYPE_ADMIN {
		return nil, model.NewAppError("UpdateUserType", "api.user.update_user_type.get.app_error", nil, "", http.StatusNotFound)
	}

	user.Type = newType

	userUpdate, err := a.Srv.Store.User().Update(user, true)
	if err != nil {
		return nil, err
	}

	ruser := userUpdate.New

	return ruser, nil
}

func (a *App) UpdatePasswordByUserIdSendEmail(userId, newPassword, method string) *model.AppError {
	user, err := a.GetUser(userId)
	if err != nil {
		return err
	}

	return a.UpdatePasswordSendEmail(user, newPassword, method)
}

func (a *App) UpdatePasswordAsUser(userId, currentPassword, newPassword string) *model.AppError {
	user, err := a.GetUser(userId)
	if err != nil {
		return err
	}

	if user == nil {
		return model.NewAppError("UpdatePasswordAsUser", "api.user.update_password_as_user.get.app_error", nil, "", http.StatusNotFound)
	}

	if err := a.DoubleCheckPassword(user, currentPassword); err != nil {
		if err.Id == "api.user.check_user_password.invalid.app_error" {
			err = model.NewAppError("updatePassword", "api.user.update_password.incorrect.app_error", nil, "", http.StatusBadRequest)
		}
		return err
	}

	return a.UpdatePasswordSendEmail(user, newPassword, "api.user.update_password.menu")
}

func (a *App) UpdatePasswordSendEmail(user *model.User, newPassword, method string) *model.AppError {
	if err := a.UpdatePassword(user, newPassword); err != nil {
		return err
	}

	a.Srv.Go(func() {
		if err := a.SendPasswordChangeCompletedEmail(user.Email, method, a.GetSiteURL()); err != nil {
			mlog.Error("Failed to send password change completed email", mlog.Err(err))
		}
	})

	return nil
}

func (a *App) UpdatePassword(user *model.User, newPassword string) *model.AppError {
	if err := a.IsPasswordValid(newPassword); err != nil {
		return err
	}

	hashedPassword := model.HashPassword(newPassword)

	if err := a.Srv.Store.User().UpdatePassword(user.Id, hashedPassword); err != nil {
		return model.NewAppError("UpdatePassword", "api.user.update_password.failed.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (a *App) ResetPasswordFromToken(userSuppliedTokenString, newPassword string) *model.AppError {
	token, err := a.GetPasswordRecoveryToken(userSuppliedTokenString)
	if err != nil {
		return err
	}
	if model.GetMillis()-token.CreateAt >= PASSWORD_RECOVER_EXPIRY_TIME {
		return model.NewAppError("resetPassword", "api.user.reset_password.link_expired.app_error", nil, "", http.StatusBadRequest)
	}

	tokenData := struct {
		UserId string
		Email  string
	}{}

	err2 := json.Unmarshal([]byte(token.Extra), &tokenData)
	if err2 != nil {
		return model.NewAppError("resetPassword", "api.user.reset_password.token_parse.error", nil, "", http.StatusInternalServerError)
	}

	user, err := a.GetUser(tokenData.UserId)
	if err != nil {
		return err
	}

	if user.Email != tokenData.Email {
		return model.NewAppError("resetPassword", "api.user.reset_password.link_expired.app_error", nil, "", http.StatusBadRequest)
	}

	if err := a.UpdatePasswordSendEmail(user, newPassword, "a reset password link"); err != nil {
		return err
	}

	if err := a.DeleteToken(token); err != nil {
		mlog.Error("Failed to delete token", mlog.Err(err))
	}

	return nil
}

func (a *App) SendPasswordReset(email string, siteURL string) *model.AppError {
	user, err := a.GetUserByEmail(email)
	if err != nil {
		return nil
	}

	token, err := a.CreatePasswordRecoveryToken(user.Id, user.Email)
	if err != nil {
		return err
	}

	return a.SendPasswordResetEmail(user.Email, token, siteURL)
}

func (a *App) CreatePasswordRecoveryToken(userId, email string) (*model.Token, *model.AppError) {
	tokenExtra := struct {
		UserId string
		Email  string
	}{
		userId,
		email,
	}

	jsonData, err := json.Marshal(tokenExtra)
	if err != nil {
		return nil, model.NewAppError("CreatePasswordRecoveryToken", "api.user.create_password_token.error", nil, "", http.StatusInternalServerError)
	}

	token := model.NewToken(TOKEN_TYPE_PASSWORD_RECOVERY, string(jsonData))

	if err := a.Srv.Store.Token().Save(token); err != nil {
		return nil, err
	}

	return token, nil
}

func (a *App) UpdateLastInboxMessageViewedForUser(message *model.InboxMessage, userId string) *model.AppError {
	return a.Srv.Store.User().UpdateLastInboxMessageViewed(message, userId)
}

func (a *App) SuspendUser(userId string, suspendSpan string, moderatorId string) *model.AppError {
	return a.Srv.Store.User().SuspendUser(userId, suspendSpan, moderatorId)
}

func (a *App) DeleteUser(userId string, sessionUserId string) *model.AppError {
	user, err := a.GetUser(userId)
	if err != nil {
		return err
	}

	if user == nil {
		return model.NewAppError("DeleteUser", "api.user.delete_user.get.app_error", nil, "", http.StatusNotFound)
	}

	if user.Type == model.USER_TYPE_ADMIN {
		return model.NewAppError("DeleteUser", "api.user.delete_admin.get.app_error", nil, "", http.StatusNotFound)
	}

	sessionUser, err := a.GetUser(sessionUserId)
	if err != nil {
		return err
	}

	if sessionUser == nil {
		return model.NewAppError("DeleteUser", "api.user.delete_user.get_session_user.app_error", nil, "", http.StatusNotFound)
	}

	// moderators can only be deleted by admin. can not self delete
	if user.Type == model.USER_TYPE_MODERATOR && sessionUser.Type != model.USER_TYPE_ADMIN {
		return model.NewAppError("DeleteUser", "api.user.delete_admin.get.app_error", nil, "", http.StatusNotFound)
	}

	// normal users can self delete
	if err := a.Srv.Store.User().Delete(userId, model.GetMillis(), sessionUserId); err != nil {
		return err
	}

	if err := a.RevokeAllSessions(user.Id); err != nil {
		return err
	}

	return nil
}

func (a *App) SetProfileImage(userId string, imageData *multipart.FileHeader) (string, *model.AppError) {
	file, err := imageData.Open()
	if err != nil {
		return "", model.NewAppError("SetProfileImage", "api.user.upload_profile_user.open.app_error", nil, err.Error(), http.StatusBadRequest)
	}
	defer file.Close()

	return a.SetProfileImageFromMultiPartFile(userId, file)
}

func (a *App) SetProfileImageFromMultiPartFile(userId string, file multipart.File) (string, *model.AppError) {
	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return "", model.NewAppError("SetProfileImage", "api.user.upload_profile_user.decode_config.app_error", nil, err.Error(), http.StatusBadRequest)
	}
	if config.Width*config.Height > model.MaxImageSize {
		return "", model.NewAppError("SetProfileImage", "api.user.upload_profile_user.too_large.app_error", nil, "", http.StatusBadRequest)
	}

	file.Seek(0, 0)

	return a.SetProfileImageFromFile(userId, file)
}

func (a *App) SetProfileImageFromFile(userId string, file io.Reader) (string, *model.AppError) {
	img, _, err := image.Decode(file)
	if err != nil {
		return "", model.NewAppError("SetProfileImage", "api.user.upload_profile_user.decode.app_error", nil, err.Error(), http.StatusBadRequest)
	}

	orientation, _ := getImageOrientation(file)
	img = makeImageUpright(img, orientation)

	// Scale profile image
	profileWidthAndHeight := 128
	img = imaging.Fill(img, profileWidthAndHeight, profileWidthAndHeight, imaging.Center, imaging.Lanczos)

	buf := new(bytes.Buffer)
	err = png.Encode(buf, img)
	if err != nil {
		return "", model.NewAppError("SetProfileImage", "api.user.upload_profile_user.encode.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	curTime := model.GetMillis()
	path := model.CreateProfileImageKey(userId, curTime)

	if err := a.WriteFile(buf, path); err != nil {
		return "", model.NewAppError("SetProfileImage", "api.user.upload_profile_user.upload_profile.app_error", nil, "", http.StatusInternalServerError)
	}

	// not save FileInfos
	if err := a.Srv.Store.User().UpdateLastPictureUpdate(userId, curTime); err != nil {
		return "", model.NewAppError("SetProfileImage", "api.user.upload_profile_user.upload_profile.app_error", nil, "", http.StatusInternalServerError)
	}

	return (*a.Config().FileSettings.AmazonCloudFrontURL + path), nil
}
