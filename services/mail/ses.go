package mail

import (
	"net/http"

	"github.com/clear-ness/qa-discussion/model"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
)

type SesMailBackend struct {
	accessKey string
	secretKey string
	region    string
}

type MailData struct {
	Sender    string
	Recipient string
	Subject   string
	HtmlBody  string
	TextBody  string
	CharSet   string
}

func NewSesMailBackend(settings *model.EmailSettings) *SesMailBackend {
	return &SesMailBackend{
		accessKey: *settings.AmazonSESAccessKeyId,
		secretKey: *settings.AmazonSESSecretAccessKey,
		region:    *settings.AmazonSESRegion,
	}
}

func (b *SesMailBackend) getSession() *session.Session {
	creds := credentials.NewStaticCredentials(b.accessKey, b.secretKey, "")
	sess, _ := session.NewSession(&aws.Config{
		Credentials: creds,
		Region:      aws.String(b.region)},
	)

	return sess
}

func (b *SesMailBackend) sesNew() *ses.SES {
	sess := b.getSession()
	sesService := ses.New(sess)

	return sesService
}

func (b *SesMailBackend) SendMail(mailData *MailData) *model.AppError {
	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			CcAddresses: []*string{},
			ToAddresses: []*string{
				aws.String(mailData.Recipient),
			},
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Html: &ses.Content{
					Charset: aws.String(mailData.CharSet),
					Data:    aws.String(mailData.HtmlBody),
				},
				Text: &ses.Content{
					Charset: aws.String(mailData.CharSet),
					Data:    aws.String(mailData.TextBody),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String(mailData.CharSet),
				Data:    aws.String(mailData.Subject),
			},
		},
		Source: aws.String(mailData.Sender),
	}

	_, err := b.sesNew().SendEmail(input)
	if err != nil {
		return model.NewAppError("SendMail", "api.mail.send_mail.ses.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}
