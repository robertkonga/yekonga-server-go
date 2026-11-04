package yekonga

import (
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	mainConfig "github.com/robertkonga/yekonga-server-go/config"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/helper/console"
)

// resizableUploadExts are the image types re-encoded to a resized .webp on upload.
var resizableUploadExts = []string{".png", ".jpg", ".jpeg", ".webp"}

// initStorage applies config.storage. With the googleDrive driver it points
// the Drive helper at the configured key file and checks the key loads, so a
// bad key shows at startup rather than on the first upload.
func (y *YekongaData) initStorage() {
	storage := y.Config.Storage

	switch storage.Driver {
	case "", mainConfig.StorageLocal:
		return
	case mainConfig.StorageGoogleDrive:
	default:
		console.Error(fmt.Sprintf("storage: unknown driver %q, using %q", storage.Driver, mainConfig.StorageLocal))
		return
	}

	if helper.IsNotEmpty(storage.GoogleDrive.KeyFile) {
		helper.GoogleDriveKeyFile = storage.GoogleDrive.KeyFile
	}
	if _, err := helper.GetGoogleDriveService(); err != nil {
		console.Error("storage:", err.Error())
	}
}

// usesGoogleDrive reports whether uploads go to Google Drive.
func (y *YekongaData) usesGoogleDrive() bool {
	return y.Config.Storage.Driver == mainConfig.StorageGoogleDrive
}

// googleDriveTenantProperty tags each tenant's Drive folder with its tenant ID.
const googleDriveTenantProperty = "yekongaTenantId"

// googleDriveFolder returns the Drive folder uploads for tenantId go into.
// Each tenant gets its own folder inside config storage.googleDrive.folderId,
// created on its first upload; uploads without a tenant go straight into
// folderId.
func (y *YekongaData) googleDriveFolder(tenantId string, host string) (string, error) {
	root := y.Config.Storage.GoogleDrive.FolderId
	if tenantId == "" {
		return root, nil
	}

	y.driveFoldersMu.RLock()
	folderId, ok := y.driveFolders[tenantId]
	y.driveFoldersMu.RUnlock()
	if ok {
		return folderId, nil
	}

	// One lookup per tenant at a time, so concurrent first uploads don't
	// each create a folder.
	v, err, _ := y.driveFolderFlight.Do(tenantId, func() (interface{}, error) {
		// The tenant's name only labels the folder; it's found again by ID.
		name := host
		if _, ok := y.models[tenantModelName]; ok {
			tenant := y.ModelQuery(tenantModelName).SkipTenant().SkipBeforeCommit().Where("id", tenantId).FindOne(nil)

			if tenantName := helper.GetValueOfString(tenant, "name"); tenantName != "" {
				name = tenantName + " (" + host + ")"
			}
		}

		folderId, err := helper.EnsureGoogleDriveFolder(name, root, map[string]string{
			googleDriveTenantProperty: tenantId,
		})
		if err != nil {
			return "", err
		}

		helper.ShareGoogleDriveFile(folderId)

		y.driveFoldersMu.Lock()
		if y.driveFolders == nil {
			y.driveFolders = map[string]string{}
		}
		y.driveFolders[tenantId] = folderId
		y.driveFoldersMu.Unlock()

		return folderId, nil
	})
	if err != nil {
		return "", err
	}

	return v.(string), nil
}

// forgetGoogleDriveFolder drops a tenant's cached folder, e.g. after it was
// deleted in Drive, so the next upload finds or recreates it.
func (y *YekongaData) forgetGoogleDriveFolder(tenantId string) {
	y.driveFoldersMu.Lock()
	delete(y.driveFolders, tenantId)
	y.driveFoldersMu.Unlock()
}

// uploadToGoogleDrive uploads a staged file into tenantId's folder. If the
// cached folder no longer exists it is looked up (or created) again once.
func (y *YekongaData) uploadToGoogleDrive(opts helper.GoogleDriveUploadOptions, tenantId string, host string) (string, error) {
	for attempt := 0; ; attempt++ {
		folderId, err := y.googleDriveFolder(tenantId, host)
		if err != nil {
			return "", err
		}

		opts.Folder = folderId
		fileId, err := helper.UploadGoogleDrive(opts)
		if err != nil && fileId == "" && attempt == 0 && tenantId != "" && helper.IsGoogleDriveNotFound(err) {
			y.forgetGoogleDriveFolder(tenantId)
			continue
		}

		return fileId, err
	}
}

// saveUploadedFile stores one uploaded file with the configured storage
// driver and returns the URL clients use to fetch it. Images are resized and
// re-encoded as .webp first. With Google Drive, the file goes into tenantId's
// own folder.
func (y *YekongaData) saveUploadedFile(fileHeader *multipart.FileHeader, host string, tenantId string) (string, error) {
	src, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	uploadDir := filepath.Join(helper.GetPath("public"), "uploads")
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		return "", err
	}

	fileExt := strings.ToLower(filepath.Ext(fileHeader.Filename))
	savedFile := filepath.Join(uploadDir, helper.GetHexString(24)+fileExt)

	dst, err := os.Create(savedFile)
	if err != nil {
		return "", fmt.Errorf("unable to save file: %w", err)
	}
	_, err = io.Copy(dst, src)
	dst.Close()
	if err != nil {
		os.Remove(savedFile)
		return "", err
	}

	originalFile := savedFile
	if helper.Contains(resizableUploadExts, fileExt) {
		targetFile := strings.TrimSuffix(savedFile, fileExt) + ".webp"
		err := helper.ResizeFile(savedFile, targetFile, helper.ResizeOptions{
			MaxWidth:     1400,
			OutputFormat: "webp",
			Quality:      80,
		})
		if err == nil {
			savedFile = targetFile
		}
	}

	if !y.usesGoogleDrive() {
		return helper.GetBaseUrl("uploads/"+filepath.Base(savedFile), host), nil
	}

	// Google Drive: the local copies are only staging files.
	if originalFile != savedFile {
		os.Remove(originalFile)
	}

	finalExt := filepath.Ext(savedFile)
	driveName := strings.TrimSuffix(filepath.Base(fileHeader.Filename), filepath.Ext(fileHeader.Filename)) + finalExt
	driveConfig := y.Config.Storage.GoogleDrive

	folder := driveConfig.FolderId

	fileID, err := y.uploadToGoogleDrive(helper.GoogleDriveUploadOptions{
		Filename: driveName,
		Filepath: savedFile,
		Folder:   folder,
	}, tenantId, host)
	if err != nil {
		if fileID == "" {
			os.Remove(savedFile)
			return "", err
		}
		// Uploaded, only the staging file could not be removed.
		console.Error(err.Error())
	}

	if driveConfig.Public {
		if _, err := helper.ShareGoogleDriveFile(fileID); err != nil {
			console.Error(err.Error())
		}
	}

	return "https://lh3.googleusercontent.com/d/" + fileID, nil
	// return helper.GetBaseUrl("download/drive/"+fileID+finalExt, host), nil
}

// serveGoogleDriveFile streams a Drive file to the client. Files are shown
// inline (so they work in <img> and links) unless ?download or ?title is
// given, which makes the browser save them; title also names the saved file.
func (y *YekongaData) serveGoogleDriveFile(req *Request, res *Response) {
	if !y.usesGoogleDrive() {
		res.Abort(http.StatusNotFound, "")
		return
	}

	fileID := req.Param("fileId")
	body, meta, err := helper.DownloadGoogleDrive(fileID)
	if err != nil {
		if helper.IsGoogleDriveNotFound(err) {
			res.Abort(http.StatusNotFound, "")
			return
		}
		console.Error(err.Error())
		res.Abort(http.StatusBadGateway, "")
		return
	}
	defer body.Close()

	name := meta.Name
	if title := req.Query("title"); title != "" {
		name = title
	}

	contentType := meta.MimeType
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(name))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	disposition := "inline"
	if req.Query("title") != "" || req.HttpRequest.URL.Query().Has("download") {
		disposition = "attachment"
	}

	res.SetHeader("content-type", contentType)
	res.SetHeader("content-disposition", fmt.Sprintf("%s; filename=%q", disposition, strings.ReplaceAll(name, `"`, "")))
	if meta.Size > 0 {
		res.SetHeader("content-length", fmt.Sprint(meta.Size))
	}
	// A Drive file ID always refers to the same uploaded content.
	res.SetHeader("cache-control", "public, max-age=31536000, immutable")
	res.SetHeader("x-content-type-options", "nosniff")

	// WriteHeader first so the body is streamed as-is, not gzipped.
	res.WriteHeader(http.StatusOK)
	if _, err := io.Copy(res, body); err != nil {
		console.Error("storage: stream drive file:", err.Error())
	}
}
