package filesstore

import (
	"bytes"
	"io"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/clear-ness/qa-discussion/model"
)

type S3FileBackend struct {
	endpoint  string
	accessKey string
	secretKey string
	secure    bool
	region    string
	bucket    string
	encrypt   string
	trace     bool
}

func NewFileBackend(settings *model.FileSettings) *S3FileBackend {
	return &S3FileBackend{
		endpoint:  *settings.AmazonS3Endpoint,
		accessKey: *settings.AmazonS3AccessKeyId,
		secretKey: *settings.AmazonS3SecretAccessKey,
		secure:    settings.AmazonS3SSL == nil || *settings.AmazonS3SSL,
		region:    *settings.AmazonS3Region,
		bucket:    *settings.AmazonS3Bucket,
	}
}

func (b *S3FileBackend) getSession() *session.Session {
	creds := credentials.NewStaticCredentials(b.accessKey, b.secretKey, "")
	sess, _ := session.NewSession(&aws.Config{
		Credentials: creds,
		Region:      aws.String(b.region)},
	)

	return sess
}

func (b *S3FileBackend) s3New() *s3.S3 {
	sess := b.getSession()
	s3Service := s3.New(sess)

	return s3Service
}

func (b *S3FileBackend) WriteFile(fr io.Reader, key string) *model.AppError {
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(fr); err != nil {
		return model.NewAppError("WriteFile", "api.file.write_file.s3.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	sess := b.getSession()
	uploader := s3manager.NewUploader(sess)

	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
		Body:   &buf,
	})
	if err != nil {
		return model.NewAppError("WriteFile", "api.file.write_file.s3.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (b *S3FileBackend) FileExists(key string) bool {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	}

	_, err := b.s3New().HeadObject(input)
	if err != nil {
		return false
	} else {
		return true
	}
}

func (b *S3FileBackend) RemoveFile(key string) *model.AppError {
	s3Service := b.s3New()
	_, err := s3Service.DeleteObject(&s3.DeleteObjectInput{Bucket: aws.String(b.bucket), Key: aws.String(key)})
	if err != nil {
		return model.NewAppError("RemoveFile", "api.file.remove_file.s3.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	if err := s3Service.WaitUntilObjectNotExists(&s3.HeadObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	}); err != nil {
		return model.NewAppError("RemoveFile", "api.file.remove_file.s3.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}
