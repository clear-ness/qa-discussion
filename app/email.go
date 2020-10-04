package app

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/clear-ness/qa-discussion/mlog"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/services/mail"
)

func emailBackend(config *model.Config) *mail.SesMailBackend {
	return mail.NewSesMailBackend(&config.EmailSettings)
}

func sendMail(mailData *mail.MailData, config *model.Config) *model.AppError {
	backend := emailBackend(config)
	return backend.SendMail(mailData)
}

func (a *App) SendWelcomeEmail(userId string, email string, verified bool, siteURL string) *model.AppError {
	mail := &mail.MailData{
		Sender:    *a.Config().EmailSettings.SupportEmail,
		Recipient: email,
		Subject:   "Welcome To QA Discussion",
		HtmlBody:  "",
		TextBody:  "",
		CharSet:   "UTF-8",
	}

	if !verified {
		token, err := a.CreateVerifyEmailToken(userId, email)
		if err != nil {
			return err
		}
		link := fmt.Sprintf("%s/sign-up/verify?token=%s&email=%s", siteURL, token.Token, url.QueryEscape(email))

		mail.HtmlBody = "<h1>Welcome to QA Discussion</h1><p>Please verify this email by checking this link: <a href=\"" + link + "\">" + link + "</a></p>"
		mail.TextBody = "Welcome to QA Discussion. Please verify this email by checking this link: " + link
	} else {
		mail.HtmlBody = "<h1>Welcome to QA Discussion</h1>"
		mail.TextBody = "Welcome to QA Discussion"
	}

	if err := sendMail(mail, a.Config()); err != nil {
		return model.NewAppError("SendWelcomeEmail", "api.user.send_welcome_email.failed.error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (a *App) SendVerifyEmail(userEmail, siteURL, token string) *model.AppError {
	link := fmt.Sprintf("%s/sign-up/verify?token=%s&email=%s", siteURL, token, url.QueryEscape(userEmail))
	htmlBody := "<p>Please verify this email by checking this link: <a href=\"" + link + "\">" + link + "</a></p>"
	textBody := "Please verify this email by checking this link: " + link

	mail := &mail.MailData{
		Sender:    *a.Config().EmailSettings.SupportEmail,
		Recipient: userEmail,
		Subject:   "QA Discussion",
		HtmlBody:  htmlBody,
		TextBody:  textBody,
		CharSet:   "UTF-8",
	}

	if err := sendMail(mail, a.Config()); err != nil {
		return model.NewAppError("SendVerifyEmail", "api.user.send_verify_email.failed.error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (a *App) SendEmailChangeEmail(oldEmail, newEmail, siteURL string) *model.AppError {
	htmlBody := "<h1>Email Changed</h1><p>Your email address of QA Discussion has been changed from " + oldEmail + " to " + newEmail + "</p>"
	textBody := "Your email address of QA Discussion has been changed from " + oldEmail + " to " + newEmail

	mail := &mail.MailData{
		Sender:    *a.Config().EmailSettings.SupportEmail,
		Recipient: newEmail,
		Subject:   "QA Discussion",
		HtmlBody:  htmlBody,
		TextBody:  textBody,
		CharSet:   "UTF-8",
	}

	if err := sendMail(mail, a.Config()); err != nil {
		return model.NewAppError("SendEmailChangeEmail", "api.user.send_email_change_email.failed.error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (a *App) SendChangeUsernameEmail(oldUsername, newUsername, email, siteURL string) *model.AppError {
	htmlBody := "<h1>Username Changed</h1><p>Your username of QA Discussion has been changed from " + oldUsername + " to " + newUsername + "</p>"
	textBody := "Your username of QA Discussion has been changed from " + oldUsername + " to " + newUsername

	mail := &mail.MailData{
		Sender:    *a.Config().EmailSettings.SupportEmail,
		Recipient: email,
		Subject:   "QA Discussion",
		HtmlBody:  htmlBody,
		TextBody:  textBody,
		CharSet:   "UTF-8",
	}

	if err := sendMail(mail, a.Config()); err != nil {
		return model.NewAppError("SendChangeUsernameEmail", "api.user.send_change_username_email.failed.error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (a *App) SendPasswordChangeCompletedEmail(email, method, siteURL string) *model.AppError {
	htmlBody := "<h1>Password Changed</h1><p>Your password of QA Discussion has been changed by " + method + "</p>"
	textBody := "Your password of QA Discussion has been changed by " + method

	mail := &mail.MailData{
		Sender:    *a.Config().EmailSettings.SupportEmail,
		Recipient: email,
		Subject:   "QA Discussion",
		HtmlBody:  htmlBody,
		TextBody:  textBody,
		CharSet:   "UTF-8",
	}

	if err := sendMail(mail, a.Config()); err != nil {
		return model.NewAppError("SendPasswordChangeCompletedEmail", "api.user.send_password_change_email.failed.error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (a *App) SendPasswordResetEmail(email string, token *model.Token, siteURL string) *model.AppError {
	link := fmt.Sprintf("%s/reset-password/complete?token=%s", siteURL, url.QueryEscape(token.Token))

	mail := &mail.MailData{
		Sender:    *a.Config().EmailSettings.SupportEmail,
		Recipient: email,
		Subject:   "QA Discussion",
		HtmlBody:  "",
		TextBody:  "",
		CharSet:   "UTF-8",
	}

	mail.HtmlBody = "<p>Please reset password by checking this link: <a href=\"" + link + "\">" + link + "</a></p>"
	mail.TextBody = "Please reset password by checking this link: " + link

	if err := sendMail(mail, a.Config()); err != nil {
		return model.NewAppError("SendPasswordResetEmail", "api.user.send_password_reset_email.failed.error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func SendInboxMessagesDigestEmail(email, siteURL string, messageCount int64, config *model.Config) *model.AppError {
	count := strconv.FormatInt(messageCount, 10)
	htmlBody := "<p>You have " + count + " unread inbox messsages since this email was last sent. Please check our site: <a href=\"" + siteURL + "\">" + siteURL + "</a></p>"
	textBody := "You have " + count + " unread inbox messages since this email was last sent, Please check our site: " + siteURL

	mail := &mail.MailData{
		Sender:    *config.EmailSettings.SupportEmail,
		Recipient: email,
		Subject:   "QA Discussion",
		HtmlBody:  htmlBody,
		TextBody:  textBody,
		CharSet:   "UTF-8",
	}

	if err := sendMail(mail, config); err != nil {
		return model.NewAppError("SendInboxMessagesDigestEmail", "api.user.send_inbox_messages_digest_email.failed.error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

// 宛先毎にTOKEN_TYPE_TEAM_INVITATIONでTokenをDBに保存し、メアド宛にsignup_user_completeを付与して送信する。
// つまりサインアップさせると同時にチーム参加API(addUserToTeamFromInvite)を叩かせる？
// TODO: メール送信(injection注意)
func (a *App) SendInviteEmails(team *model.Team, senderName string, senderUserId string, invites []string, siteURL string) {
	for _, invite := range invites {
		if len(invite) > 0 {
			//subject := utils.T("api.templates.invite_subject",
			//    map[string]interface{}{"SenderName": senderName,
			//        "TeamDisplayName": team.DisplayName,
			//        "SiteName":        a.ClientConfig()["SiteName"]})

			//bodyPage := a.newEmailTemplate("invite_body", model.DEFAULT_LOCALE)
			//bodyPage.Props["SiteURL"] = siteURL
			//bodyPage.Props["Title"] = utils.T("api.templates.invite_body.title")
			//bodyPage.Html["Info"] = utils.TranslateAsHtml(utils.T, "api.templates.invite_body.info",
			//    map[string]interface{}{"SenderName": senderName, "TeamDisplayName": team.DisplayName})
			//bodyPage.Props["Button"] = utils.T("api.templates.invite_body.button")
			//bodyPage.Html["ExtraInfo"] = utils.TranslateAsHtml(utils.T, "api.templates.invite_body.extra_info",
			//    map[string]interface{}{"TeamDisplayName": team.DisplayName})
			//bodyPage.Props["TeamURL"] = siteURL + "/" + team.Name

			token := model.NewToken(
				TOKEN_TYPE_TEAM_INVITATION,
				model.MapToJson(map[string]string{"teamId": team.Id, "email": invite}),
			)

			//props := make(map[string]string)
			//props["email"] = invite
			//props["name"] = team.Name
			//data := model.MapToJson(props)

			if err := a.Srv.Store.Token().Save(token); err != nil {
				mlog.Error("Failed to send invite email successfully ", mlog.Err(err))
				continue
			}
			//bodyPage.Props["Link"] = fmt.Sprintf("%s/signup_user_complete/?d=%s&t=%s", siteURL, url.QueryEscape(data), url.QueryEscape(token.Token))

			//if err := a.sendMail(invite, subject, bodyPage.Render()); err != nil {
			//    mlog.Error("Failed to send invite email successfully ", mlog.Err(err))
			//}
		}
	}
}
