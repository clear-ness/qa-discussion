package api

import (
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"time"

	"github.com/clear-ness/qa-discussion/app"
	"github.com/clear-ness/qa-discussion/model"
)

const (
	maxUploadDrainBytes = (10 * 1024 * 1024) // 10Mb
)

func (api *API) InitFile() {
	api.BaseRoutes.Files.Handle("", api.ApiSessionRequired(uploadFileStream)).Methods("POST")

	api.BaseRoutes.File.Handle("/info", api.ApiHandler(getFileInfo)).Methods("GET")
}

func parseMultipartRequestHeader(req *http.Request) (boundary string, err error) {
	v := req.Header.Get("Content-Type")
	if v == "" {
		return "", http.ErrNotMultipart
	}
	d, params, err := mime.ParseMediaType(v)
	if err != nil || d != "multipart/form-data" {
		return "", http.ErrNotMultipart
	}
	boundary, ok := params["boundary"]
	if !ok {
		return "", http.ErrMissingBoundary
	}

	return boundary, nil
}

func uploadFileStream(c *Context, w http.ResponseWriter, r *http.Request) {
	// Drain any remaining bytes in the request body, up to a limit
	defer io.CopyN(ioutil.Discard, r.Body, maxUploadDrainBytes)

	timestamp := time.Now()
	var fileUploadResponse *model.FileUploadResponse

	_, err := parseMultipartRequestHeader(r)
	switch err {
	case http.ErrNotMultipart:
		fileUploadResponse = uploadFileSimple(c, r, timestamp)
	default:
		c.Err = model.NewAppError("uploadFileStream",
			"api.file.upload_file.read_request.app_error",
			nil, err.Error(), http.StatusBadRequest)
	}
	if c.Err != nil {
		return
	}

	// Write the response values to the output upon return
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fileUploadResponse.ToJson()))
}

// uploads a file from a simple POST with the file in the request body
func uploadFileSimple(c *Context, r *http.Request, timestamp time.Time) *model.FileUploadResponse {
	// Simple POST with the file in the body and all metadata in the args.
	c.RequireFilename()
	if c.Err != nil {
		return nil
	}

	clientId := r.Form.Get("client_id")
	info, appErr := c.App.UploadFileX(c.Params.Filename, r.Body,
		app.UploadFileSetUserId(c.App.Session.UserId),
		app.UploadFileSetTimestamp(timestamp),
		app.UploadFileSetContentLength(r.ContentLength),
		app.UploadFileSetClientId(clientId))
	if appErr != nil {
		c.Err = appErr
		return nil
	}

	fileUploadResponse := &model.FileUploadResponse{
		FileInfos: []*model.FileInfo{info},
	}
	if clientId != "" {
		fileUploadResponse.ClientIds = []string{clientId}
	}
	return fileUploadResponse
}

func getFileInfo(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireFileId()
	if c.Err != nil {
		return
	}

	info, err := c.App.GetFileInfo(c.Params.FileId)
	if err != nil {
		c.Err = err
		return
	}

	w.Header().Set("Cache-Control", "max-age=2592000, public")
	w.Write([]byte(info.ToJson()))
}
