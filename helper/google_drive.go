package helper

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/robertkonga/yekonga-server-go/helper/console"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

// GoogleDriveKeyFile is the path to the Google service-account JSON key used
// for Drive calls. Relative paths are looked up next to the executable, then
// in the working directory.
// Change it before the first Drive call; the service is built once per path.
var GoogleDriveKeyFile = "service-keys.json"

const googleDriveFolderMimeType = "application/vnd.google-apps.folder"

var (
	googleDriveMu      sync.Mutex
	googleDriveService *drive.Service
	googleDriveKeyPath string
)

// GoogleDriveUploadOptions describes a file to upload to Google Drive.
type GoogleDriveUploadOptions struct {
	Filename string // Name of the file in Drive; defaults to the base name of Filepath
	Filepath string // Local file to upload
	Folder   string // Parent folder ID; empty uploads to the account's root
	MimeType string // Defaults to the type implied by the file extension, else image/jpeg
	KeepFile bool   // Keep the local file after a successful upload (it is removed by default)
}

// GetGoogleDriveService returns a Drive v3 client authenticated with the
// service-account key at GoogleDriveKeyFile.
func GetGoogleDriveService() (*drive.Service, error) {
	googleDriveMu.Lock()
	defer googleDriveMu.Unlock()

	keyPath := GoogleDriveKeyFile
	if googleDriveService != nil && googleDriveKeyPath == keyPath {
		return googleDriveService, nil
	}

	resolvedPath := GetPath(keyPath)
	console.Error("google drive: GoogleDriveKeyFile", resolvedPath)

	keyJSON, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("google drive: read key file %q: %w", resolvedPath, err)
	}

	service, err := drive.NewService(
		context.Background(),
		// option.WithCredentialsJSON(keyJSON),
		option.WithAuthCredentialsJSON(option.ServiceAccount, keyJSON),
		option.WithScopes(drive.DriveScope),
	)
	if err != nil {
		return nil, fmt.Errorf("google drive: create service: %w", err)
	}

	googleDriveService = service
	googleDriveKeyPath = keyPath
	return service, nil
}

// UploadGoogleDrive uploads a local file to Google Drive and returns the new
// file's ID. Unless KeepFile is set, the local file is deleted after a
// successful upload.
func UploadGoogleDrive(opts GoogleDriveUploadOptions) (string, error) {
	if opts.Filepath == "" {
		return "", fmt.Errorf("google drive: filepath is required")
	}

	service, err := GetGoogleDriveService()
	if err != nil {
		return "", err
	}

	filename := opts.Filename
	if filename == "" {
		filename = filepath.Base(opts.Filepath)
	}

	mimeType := opts.MimeType
	if mimeType == "" {
		mimeType = mime.TypeByExtension(filepath.Ext(opts.Filepath))
	}
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	f, err := os.Open(opts.Filepath)
	if err != nil {
		return "", fmt.Errorf("google drive: open %q: %w", opts.Filepath, err)
	}

	metadata := &drive.File{Name: filename, MimeType: mimeType}
	if opts.Folder != "" {
		metadata.Parents = []string{opts.Folder}
	}

	created, err := service.Files.Create(metadata).
		Media(f).
		Fields("id", "name").
		SupportsAllDrives(true).
		Do()
	f.Close()
	if err != nil {
		return "", fmt.Errorf("google drive: upload %q: %w", filename, err)
	}

	if !opts.KeepFile {
		if err := os.Remove(opts.Filepath); err != nil {
			return created.Id, fmt.Errorf("google drive: uploaded but could not remove %q: %w", opts.Filepath, err)
		}
	}

	return created.Id, nil
}

// CreateGoogleDriveFolder creates a folder in Google Drive and returns its ID.
// Pass a parent folder ID to nest it; omit it to create the folder at the root.
func CreateGoogleDriveFolder(name string, parent ...string) (string, error) {
	service, err := GetGoogleDriveService()
	if err != nil {
		return "", err
	}

	metadata := &drive.File{Name: name, MimeType: googleDriveFolderMimeType}
	if len(parent) > 0 && parent[0] != "" {
		metadata.Parents = []string{parent[0]}
	}

	created, err := service.Files.Create(metadata).
		Fields("id").
		SupportsAllDrives(true).
		Do()

	if err != nil {
		return "", fmt.Errorf("google drive: create folder %q: %w", name, err)
	}

	return created.Id, nil
}

// EnsureGoogleDriveFolder returns the ID of the folder inside parent whose
// appProperties include all of properties, creating it (named name, tagged
// with properties) when there is none. An empty parent means the account's
// root. The folder is found by its properties, not its name, so renaming it
// in Drive doesn't matter. If several match, the oldest wins.
func EnsureGoogleDriveFolder(name, parent string, properties map[string]string) (string, error) {
	if len(properties) == 0 {
		return "", fmt.Errorf("google drive: folder properties are required")
	}

	service, err := GetGoogleDriveService()
	if err != nil {
		return "", err
	}

	if parent == "" {
		parent = "root"
	}

	keys := make([]string, 0, len(properties))
	for k := range properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	query := []string{
		fmt.Sprintf("'%s' in parents", googleDriveQuote(parent)),
		fmt.Sprintf("mimeType = '%s'", googleDriveFolderMimeType),
		"trashed = false",
	}
	for _, k := range keys {
		query = append(query, fmt.Sprintf("appProperties has { key='%s' and value='%s' }",
			googleDriveQuote(k), googleDriveQuote(properties[k])))
	}

	list, err := service.Files.List().
		Q(strings.Join(query, " and ")).
		Fields("files(id)").
		OrderBy("createdTime").
		PageSize(1).
		Corpora("allDrives").
		IncludeItemsFromAllDrives(true).
		SupportsAllDrives(true).
		Do()
	if err != nil {
		return "", fmt.Errorf("google drive: find folder %q: %w", name, err)
	}
	if len(list.Files) > 0 {
		return list.Files[0].Id, nil
	}

	metadata := &drive.File{
		Name:          name,
		MimeType:      googleDriveFolderMimeType,
		Parents:       []string{parent},
		AppProperties: properties,
	}
	created, err := service.Files.Create(metadata).
		Fields("id").
		SupportsAllDrives(true).
		Do()
	if err != nil {
		return "", fmt.Errorf("google drive: create folder %q: %w", name, err)
	}

	return created.Id, nil
}

// googleDriveQuote escapes a value for a single-quoted Drive query string.
func googleDriveQuote(s string) string {
	return strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(s)
}

// ShareGoogleDriveFile makes a file readable by anyone with the link and
// returns the IDs of the permissions it created.
func ShareGoogleDriveFile(fileID string) ([]string, error) {
	service, err := GetGoogleDriveService()
	if err != nil {
		return nil, err
	}

	permissions := []*drive.Permission{
		{Type: "anyone", Role: "reader"},
	}

	ids := make([]string, 0, len(permissions))
	for _, permission := range permissions {
		result, err := service.Permissions.Create(fileID, permission).
			Fields("id").
			SupportsAllDrives(true).
			Do()
		if err != nil {
			return ids, fmt.Errorf("google drive: share %q: %w", fileID, err)
		}
		ids = append(ids, result.Id)
	}

	return ids, nil
}

// DownloadGoogleDrive opens a Drive file's content for reading and returns it
// with the file's metadata (id, name, mimeType, size). The caller must close
// the returned reader.
func DownloadGoogleDrive(fileID string) (io.ReadCloser, *drive.File, error) {
	service, err := GetGoogleDriveService()
	if err != nil {
		return nil, nil, err
	}

	meta, err := service.Files.Get(fileID).
		Fields("id", "name", "mimeType", "size").
		SupportsAllDrives(true).
		Do()
	if err != nil {
		return nil, nil, fmt.Errorf("google drive: get %q: %w", fileID, err)
	}

	resp, err := service.Files.Get(fileID).SupportsAllDrives(true).Download()
	if err != nil {
		return nil, nil, fmt.Errorf("google drive: download %q: %w", fileID, err)
	}

	return resp.Body, meta, nil
}

// IsGoogleDriveNotFound reports whether err is Drive's "file not found".
func IsGoogleDriveNotFound(err error) bool {
	var apiErr *googleapi.Error
	return errors.As(err, &apiErr) && apiErr.Code == http.StatusNotFound
}
