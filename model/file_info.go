package model

import (
	"encoding/json"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

type FileInfo struct {
	Id            string `db:"Id, primarykey" json:"id"`
	UserId        string `db:"UserId" json:"user_id"`
	PostId        string `db:"PostId" json:"post_id,omitempty"`
	CreateAt      int64  `db:"CreateAt" json:"create_at"`
	DeleteAt      int64  `db:"DeleteAt" json:"delete_at"`
	Path          string `db:"Path" json:"-"`
	ThumbnailPath string `db:"ThumbnailPath" json:"-"`
	Name          string `db:"Name" json:"name"`
	Extension     string `db:"Extension" json:"extension"`
	Size          int64  `db:"Size" json:"size"`
	MimeType      string `db:"MimeType" json:"mime_type"`
	Width         int    `db:"Width" json:"width,omitempty"`
	Height        int    `db:"Height" json:"height,omitempty"`
	// → width, height is pixel

	// Link is for end users
	Link          string `db:"-" json:"link,omitempty"`
	ThumbnailLink string `db:"-" json:"thumbnail_link,omitempty"`
}

func (o *FileInfo) ToJson() string {
	b, _ := json.Marshal(o)
	return string(b)
}

func (o *FileInfo) PreSave() {
	if o.Id == "" {
		o.Id = NewId()
	}

	if o.CreateAt == 0 {
		o.CreateAt = GetMillis()
	}
}

func (o *FileInfo) IsValid() *AppError {
	if len(o.Id) != 26 {
		return NewAppError("FileInfo.IsValid", "model.file_info.is_valid.id.app_error", nil, "", http.StatusBadRequest)
	}

	if len(o.UserId) != 26 {
		return NewAppError("FileInfo.IsValid", "model.file_info.is_valid.user_id.app_error", nil, "id="+o.Id, http.StatusBadRequest)
	}

	if len(o.PostId) != 0 && len(o.PostId) != 26 {
		return NewAppError("FileInfo.IsValid", "model.file_info.is_valid.post_id.app_error", nil, "id="+o.Id, http.StatusBadRequest)
	}

	if o.CreateAt == 0 {
		return NewAppError("FileInfo.IsValid", "model.file_info.is_valid.create_at.app_error", nil, "id="+o.Id, http.StatusBadRequest)
	}

	if o.Path == "" {
		return NewAppError("FileInfo.IsValid", "model.file_info.is_valid.path.app_error", nil, "id="+o.Id, http.StatusBadRequest)
	}

	return nil
}

func (o *FileInfo) IsImage() bool {
	return strings.HasPrefix(o.MimeType, "image")
}

func NewInfo(name string) *FileInfo {
	info := &FileInfo{
		Name: name,
	}

	extension := strings.ToLower(filepath.Ext(name))
	info.MimeType = mime.TypeByExtension(extension)

	if extension != "" && extension[0] == '.' {
		info.Extension = extension[1:]
	} else {
		info.Extension = extension
	}

	return info
}

func (o *FileInfo) SetLinksForClient(settings *FileSettings) *FileInfo {
	if o.Path != "" {
		o.Link = *settings.AmazonCloudFrontURL + o.Path
	}

	if o.ThumbnailPath != "" {
		o.ThumbnailLink = *settings.AmazonCloudFrontURL + o.ThumbnailPath
	}

	return o
}
