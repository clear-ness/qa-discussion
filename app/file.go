package app

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/clear-ness/qa-discussion/mlog"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/services/filesstore"

	"github.com/disintegration/imaging"
	"github.com/rwcarlsen/goexif/exif"
)

const (
	Upright            = 1
	UprightMirrored    = 2
	UpsideDown         = 3
	UpsideDownMirrored = 4
	RotatedCWMirrored  = 5
	RotatedCCW         = 6
	RotatedCCWMirrored = 7
	RotatedCW          = 8

	MaxImageSize         = 6048 * 4032 // 24 megapixels, roughly 36MB as a raw image
	ImageThumbnailWidth  = 120
	ImageThumbnailHeight = 100
	ImageThumbnailRatio  = float64(ImageThumbnailHeight) / float64(ImageThumbnailWidth)

	UploadFileInitialBufferSize = 2 * 1024 * 1024 // 2Mb
)

func (a *App) FileBackend() *filesstore.S3FileBackend {
	return filesstore.NewFileBackend(&a.Config().FileSettings)
}

func (a *App) WriteFile(fr io.Reader, path string) *model.AppError {
	backend := a.FileBackend()
	return backend.WriteFile(fr, path)
}

func (a *App) FileExists(path string) bool {
	backend := a.FileBackend()
	return backend.FileExists(path)
}

func (a *App) RemoveFile(path string) *model.AppError {
	backend := a.FileBackend()
	return backend.RemoveFile(path)
}

func getImageOrientation(input io.Reader) (int, error) {
	exifData, err := exif.Decode(input)
	if err != nil {
		return Upright, err
	}

	tag, err := exifData.Get("Orientation")
	if err != nil {
		return Upright, err
	}

	orientation, err := tag.Int(0)
	if err != nil {
		return Upright, err
	}

	return orientation, nil
}

func makeImageUpright(img image.Image, orientation int) image.Image {
	switch orientation {
	case UprightMirrored:
		return imaging.FlipH(img)
	case UpsideDown:
		return imaging.Rotate180(img)
	case UpsideDownMirrored:
		return imaging.FlipV(img)
	case RotatedCWMirrored:
		return imaging.Transpose(img)
	case RotatedCCW:
		return imaging.Rotate270(img)
	case RotatedCCWMirrored:
		return imaging.Transverse(img)
	case RotatedCW:
		return imaging.Rotate90(img)
	default:
		return img
	}
}

func UploadFileSetUserId(userId string) func(t *uploadFileTask) {
	return func(t *uploadFileTask) {
		t.UserId = filepath.Base(userId)
	}
}

func UploadFileSetTimestamp(timestamp time.Time) func(t *uploadFileTask) {
	return func(t *uploadFileTask) {
		t.Timestamp = timestamp
	}
}

func UploadFileSetContentLength(contentLength int64) func(t *uploadFileTask) {
	return func(t *uploadFileTask) {
		t.ContentLength = contentLength
	}
}

func UploadFileSetClientId(clientId string) func(t *uploadFileTask) {
	return func(t *uploadFileTask) {
		t.ClientId = clientId
	}
}

type uploadFileTask struct {
	// File name.
	Name string

	UserId string

	// Time stamp to use when creating the file.
	Timestamp time.Time

	// The value of the Content-Length http header, when available.
	ContentLength int64

	// The file data stream.
	Input io.Reader

	// An optional, client-assigned Id field.
	// Not FileInfo.Id.
	ClientId string

	//=============================================================
	// Internal state
	buf          *bytes.Buffer
	limit        int64
	limitedInput io.Reader
	teeInput     io.Reader
	fileinfo     *model.FileInfo
	maxFileSize  int64

	// Cached image data that (may) get initialized in preprocessImage and
	// is used in postprocessImage
	decoded          image.Image
	imageType        string
	imageOrientation int

	// Testing: overrideable dependency functions
	writeFile      func(io.Reader, string) *model.AppError
	saveToDatabase func(*model.FileInfo) (*model.FileInfo, *model.AppError)
}

func (t *uploadFileTask) init(a *App) {
	t.buf = &bytes.Buffer{}
	t.maxFileSize = *a.Config().FileSettings.MaxFileSize
	t.limit = *a.Config().FileSettings.MaxFileSize

	t.fileinfo = model.NewInfo(filepath.Base(t.Name))
	t.fileinfo.Id = model.NewId()
	t.fileinfo.UserId = t.UserId
	t.fileinfo.CreateAt = t.Timestamp.UnixNano() / int64(time.Millisecond)
	t.fileinfo.Path = t.pathPrefix() + t.Name

	// Prepare to read ContentLength if it is known, otherwise limit
	// ourselves to MaxFileSize. Add an extra byte to check and fail if the
	// client sent too many bytes.
	if t.ContentLength > 0 {
		t.limit = t.ContentLength
		// Over-Grow the buffer to prevent bytes.ReadFrom from doing it
		// at the very end.
		t.buf.Grow(int(t.limit + 1 + bytes.MinRead))
	} else {
		// If we don't know the upload size, grow the buffer somewhat
		// anyway to avoid extra reslicing.
		t.buf.Grow(UploadFileInitialBufferSize)
	}
	t.limitedInput = &io.LimitedReader{
		R: t.Input,
		N: t.limit + 1,
	}
	t.teeInput = io.TeeReader(t.limitedInput, t.buf)

	t.writeFile = a.WriteFile
	t.saveToDatabase = a.Srv.Store.FileInfo().Save
}

func (a *App) UploadFileX(name string, input io.Reader,
	opts ...func(*uploadFileTask)) (*model.FileInfo, *model.AppError) {

	t := &uploadFileTask{
		Name:  filepath.Base(name),
		Input: input,
	}
	for _, o := range opts {
		o(t)
	}
	t.init(a)

	if t.ContentLength > t.maxFileSize {
		return nil, t.newAppError("api.file.upload_file.too_large_detailed.app_error",
			"", http.StatusRequestEntityTooLarge, "Length", t.ContentLength, "Limit", t.maxFileSize)
	}

	var aerr *model.AppError
	if t.fileinfo.IsImage() {
		aerr = t.preprocessImage()
		if aerr != nil {
			return t.fileinfo, aerr
		}
	}

	aerr = t.readAll()
	if aerr != nil {
		return t.fileinfo, aerr
	}

	// Concurrently upload and update DB, and post-process the image.
	wg := sync.WaitGroup{}

	if t.fileinfo.IsImage() {
		wg.Add(1)
		go func() {
			// create thumbail image, and upload it to aws S3
			t.postprocessImage()
			wg.Done()
		}()
	}

	// upload the original image to aws S3
	aerr = t.writeFile(t.newReader(), t.fileinfo.Path)
	if aerr != nil {
		return nil, aerr
	}

	if _, err := t.saveToDatabase(t.fileinfo); err != nil {
		return nil, err
	}

	wg.Wait()

	t.fileinfo.SetLinksForClient(&a.Config().FileSettings)

	return t.fileinfo, nil
}

func (t uploadFileTask) newReader() io.Reader {
	if t.teeInput != nil {
		return io.MultiReader(bytes.NewReader(t.buf.Bytes()), t.teeInput)
	} else {
		return bytes.NewReader(t.buf.Bytes())
	}
}

func (t *uploadFileTask) readAll() *model.AppError {
	_, err := t.buf.ReadFrom(t.limitedInput)
	if err != nil {
		return t.newAppError("api.file.upload_file.read_request.app_error",
			err.Error(), http.StatusBadRequest)
	}
	if int64(t.buf.Len()) > t.limit {
		return t.newAppError("api.file.upload_file.too_large_detailed.app_error",
			"", http.StatusRequestEntityTooLarge, "Length", t.buf.Len(), "Limit", t.limit)
	}
	t.fileinfo.Size = int64(t.buf.Len())

	t.limitedInput = nil
	t.teeInput = nil
	return nil
}

func (t uploadFileTask) pathPrefix() string {
	return "/users/" + t.UserId + "/" + t.fileinfo.Id + "/"
}

func (t uploadFileTask) newAppError(id string, details interface{}, httpStatus int, extra ...interface{}) *model.AppError {
	params := map[string]interface{}{
		"Name":          t.Name,
		"Filename":      t.Name,
		"UserId":        t.UserId,
		"ContentLength": t.ContentLength,
		"ClientId":      t.ClientId,
	}
	if t.fileinfo != nil {
		params["Width"] = t.fileinfo.Width
		params["Height"] = t.fileinfo.Height
	}
	for i := 0; i+1 < len(extra); i += 2 {
		params[fmt.Sprintf("%v", extra[i])] = extra[i+1]
	}

	return model.NewAppError("uploadFileTask", id, params, fmt.Sprintf("%v", details), httpStatus)
}

func (t *uploadFileTask) preprocessImage() *model.AppError {
	// If SVG, attempt to extract dimensions and then return
	if t.fileinfo.MimeType == "image/svg+xml" {
		svgInfo, err := parseSVG(t.newReader())
		if err != nil {
			mlog.Error("Failed to parse SVG", mlog.Err(err))
		}
		if svgInfo.Width > 0 && svgInfo.Height > 0 {
			t.fileinfo.Width = svgInfo.Width
			t.fileinfo.Height = svgInfo.Height
		}
		return nil
	}

	// If we fail to decode, return "as is".
	config, _, err := image.DecodeConfig(t.newReader())
	if err != nil {
		return nil
	}

	t.fileinfo.Width = config.Width
	t.fileinfo.Height = config.Height

	// Check dimensions before loading the whole thing into memory later on.
	if t.fileinfo.Width*t.fileinfo.Height > MaxImageSize {
		return t.newAppError("api.file.upload_file.large_image_detailed.app_error",
			"", http.StatusBadRequest)
	}

	nameWithoutExtension := t.Name[:strings.LastIndex(t.Name, ".")]
	t.fileinfo.ThumbnailPath = t.pathPrefix() + nameWithoutExtension + "_thumb.jpg"

	// check the image orientation with goexif; consume the bytes we
	// already have first, then keep Tee-ing from input.
	// TODO: try to reuse exif's .Raw buffer rather than Tee-ing
	if t.imageOrientation, err = getImageOrientation(t.newReader()); err == nil &&
		(t.imageOrientation == RotatedCWMirrored ||
			t.imageOrientation == RotatedCCW ||
			t.imageOrientation == RotatedCCWMirrored ||
			t.imageOrientation == RotatedCW) {
		t.fileinfo.Width, t.fileinfo.Height = t.fileinfo.Height, t.fileinfo.Width
	}

	// For animated GIFs disable the preview; since we have to Decode gifs
	// anyway, cache the decoded image for later.
	if t.fileinfo.MimeType == "image/gif" {
		gifConfig, err := gif.DecodeAll(t.newReader())
		if err == nil {
			if len(gifConfig.Image) > 0 {
				t.decoded = gifConfig.Image[0]
				t.imageType = "gif"
			}
		}
	}

	return nil
}

func (t *uploadFileTask) postprocessImage() {
	// don't try to process SVG files
	if t.fileinfo.MimeType == "image/svg+xml" {
		return
	}

	decoded, typ := t.decoded, t.imageType
	if decoded == nil {
		var err error
		decoded, typ, err = image.Decode(t.newReader())
		if err != nil {
			mlog.Error("Unable to decode image", mlog.Err(err))
			return
		}
	}

	// Fill in the background of a potentially-transparent png file as
	// white.
	if typ == "png" {
		dst := image.NewRGBA(decoded.Bounds())
		draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
		draw.Draw(dst, dst.Bounds(), decoded, decoded.Bounds().Min, draw.Over)
		decoded = dst
	}

	decoded = makeImageUpright(decoded, t.imageOrientation)
	if decoded == nil {
		return
	}

	writeJPEG := func(img image.Image, path string) {
		// That is, each Write to the 'w' blocks until it has satisfied
		// one or more Reads from the 'r' that fully consume
		// the written data.
		r, w := io.Pipe()
		go func() {
			// upload the modified image
			aerr := t.writeFile(r, path)
			if aerr != nil {
				mlog.Error("Unable to upload", mlog.String("path", path), mlog.Err(aerr))
				return
			}
		}()

		err := jpeg.Encode(w, img, &jpeg.Options{Quality: 90})
		if err != nil {
			mlog.Error("Unable to encode image as jpeg", mlog.String("path", path), mlog.Err(err))
			w.CloseWithError(err)
		} else {
			w.Close()
		}
	}

	w := decoded.Bounds().Dx()
	h := decoded.Bounds().Dy()

	// create thumbnail image, and upload it
	go func() {
		thumb := decoded
		if h > ImageThumbnailHeight || w > ImageThumbnailWidth {
			if float64(h)/float64(w) < ImageThumbnailRatio {
				thumb = imaging.Resize(decoded, 0, ImageThumbnailHeight, imaging.Lanczos)
			} else {
				thumb = imaging.Resize(decoded, ImageThumbnailWidth, 0, imaging.Lanczos)
			}
		}
		writeJPEG(thumb, t.fileinfo.ThumbnailPath)
	}()
}

func (a *App) GetFileInfo(fileId string) (*model.FileInfo, *model.AppError) {
	info, err := a.Srv.Store.FileInfo().Get(fileId)
	if err != nil {
		return nil, err
	}

	info.SetLinksForClient(&a.Config().FileSettings)

	return info, nil
}
