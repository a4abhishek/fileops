package filesystem

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/a4abhishek/fileops/pkg/domain"
	"github.com/cespare/xxhash/v2"
	"golang.org/x/crypto/blake2b"
)

// OSFileSystem implements the FileSystem interface using the operating system
type OSFileSystem struct {
	chunkSize int64
}

// NewOSFileSystem creates a new OS-based file system implementation
func NewOSFileSystem(chunkSize int64) *OSFileSystem {
	if chunkSize <= 0 {
		chunkSize = 64 * 1024 * 1024 // 64MB default
	}
	return &OSFileSystem{
		chunkSize: chunkSize,
	}
}

// Walk traverses the file system starting from the given path
func (fs *OSFileSystem) Walk(ctx context.Context, path string, fn domain.WalkFunc) error {
	return filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var fileInfo *domain.FileInfo
		if info != nil {
			fileInfo = &domain.FileInfo{
				Path:    filePath,
				Name:    info.Name(),
				Size:    info.Size(),
				ModTime: info.ModTime(),
				IsDir:   info.IsDir(),
				Mode:    uint32(info.Mode()),
			}
		}

		return fn(filePath, fileInfo, err)
	})
}

// Stat returns file information for the given path
func (fs *OSFileSystem) Stat(path string) (*domain.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	return &domain.FileInfo{
		Path:    path,
		Name:    info.Name(),
		Size:    info.Size(),
		ModTime: info.ModTime(),
		IsDir:   info.IsDir(),
		Mode:    uint32(info.Mode()),
	}, nil
}

// Remove removes the file or directory at the given path
func (fs *OSFileSystem) Remove(path string) error {
	return os.Remove(path)
}

// RemoveAll removes the directory and all its contents
func (fs *OSFileSystem) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

// Move moves a file or directory from source to destination
func (fs *OSFileSystem) Move(source, destination string) error {
	return os.Rename(source, destination)
}

// Copy copies a file or directory from source to destination
func (fs *OSFileSystem) Copy(source, destination string) error {
	sourceInfo, err := os.Stat(source)
	if err != nil {
		return err
	}

	if sourceInfo.IsDir() {
		return fs.copyDir(source, destination)
	}
	return fs.copyFile(source, destination)
}

// copyFile copies a single file
func (fs *OSFileSystem) copyFile(source, destination string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create destination directory if it doesn't exist
	destDir := filepath.Dir(destination)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	destFile, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy file content
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(source)
	if err != nil {
		return err
	}

	return os.Chmod(destination, sourceInfo.Mode())
}

// copyDir copies a directory and all its contents
func (fs *OSFileSystem) copyDir(source, destination string) error {
	sourceInfo, err := os.Stat(source)
	if err != nil {
		return err
	}

	// Create destination directory
	if err := os.MkdirAll(destination, sourceInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		sourcePath := filepath.Join(source, entry.Name())
		destPath := filepath.Join(destination, entry.Name())

		if entry.IsDir() {
			if err := fs.copyDir(sourcePath, destPath); err != nil {
				return err
			}
		} else {
			if err := fs.copyFile(sourcePath, destPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// CreateDir creates a directory at the given path
func (fs *OSFileSystem) CreateDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// IsEmpty checks if a directory is empty
func (fs *OSFileSystem) IsEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

// Exists checks if a file or directory exists
func (fs *OSFileSystem) Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// ComputeHash computes the hash of a file using the specified algorithm
func (fs *OSFileSystem) ComputeHash(path string, algorithm string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var hasher hash.Hash

	switch strings.ToLower(algorithm) {
	case "md5":
		hasher = md5.New()
	case "sha1":
		hasher = sha1.New()
	case "sha256":
		hasher = sha256.New()
	case "sha512":
		hasher = sha512.New()
	case "blake2b":
		var err error
		hasher, err = blake2b.New256(nil)
		if err != nil {
			return "", err
		}
	case "xxhash64":
		hasher = xxhash.New()
	case "crc32":
		hasher = crc32.NewIEEE()
	default:
		return "", fmt.Errorf("unsupported hash algorithm: %s", algorithm)
	}

	// Stream the file content to the hasher in chunks
	buffer := make([]byte, fs.chunkSize)
	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return "", err
		}
		if n == 0 {
			break
		}

		if _, err := hasher.Write(buffer[:n]); err != nil {
			return "", err
		}
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// PathValidator provides utilities for validating and normalizing paths
type PathValidator struct {
	excludePatterns []string
	includePatterns []string
}

// NewPathValidator creates a new path validator
func NewPathValidator(excludePatterns, includePatterns []string) *PathValidator {
	return &PathValidator{
		excludePatterns: excludePatterns,
		includePatterns: includePatterns,
	}
}

// IsValid checks if a path should be processed based on include/exclude patterns
func (pv *PathValidator) IsValid(path string) bool {
	// Check exclude patterns first
	for _, pattern := range pv.excludePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return false
		}
		// Also check if the pattern matches any part of the path
		if strings.Contains(strings.ToLower(path), strings.ToLower(pattern)) {
			return false
		}
	}

	// If no include patterns specified, allow all (except excluded)
	if len(pv.includePatterns) == 0 {
		return true
	}

	// Check include patterns
	for _, pattern := range pv.includePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
		// Also check if the pattern matches any part of the path
		if strings.Contains(strings.ToLower(path), strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// Normalize normalizes a file path for the current operating system
func (pv *PathValidator) Normalize(path string) string {
	// Clean the path
	cleaned := filepath.Clean(path)

	// Convert to absolute path if relative
	if !filepath.IsAbs(cleaned) {
		if abs, err := filepath.Abs(cleaned); err == nil {
			cleaned = abs
		}
	}

	return cleaned
}

// IsHidden checks if a file or directory is hidden
func (pv *PathValidator) IsHidden(path string) bool {
	name := filepath.Base(path)
	// Unix-style hidden files (starting with .)
	if strings.HasPrefix(name, ".") && name != "." && name != ".." {
		return true
	}

	// TODO: Add Windows-specific hidden file detection using file attributes
	return false
}

// FileTypeDetector provides utilities for detecting file types
type FileTypeDetector struct {
	mimeTypes map[string]string
}

// NewFileTypeDetector creates a new file type detector
func NewFileTypeDetector() *FileTypeDetector {
	return &FileTypeDetector{
		mimeTypes: map[string]string{
			".txt":  "text/plain",
			".md":   "text/markdown",
			".html": "text/html",
			".htm":  "text/html",
			".css":  "text/css",
			".js":   "application/javascript",
			".json": "application/json",
			".xml":  "application/xml",
			".pdf":  "application/pdf",
			".zip":  "application/zip",
			".tar":  "application/x-tar",
			".gz":   "application/gzip",
			".jpg":  "image/jpeg",
			".jpeg": "image/jpeg",
			".png":  "image/png",
			".gif":  "image/gif",
			".bmp":  "image/bmp",
			".svg":  "image/svg+xml",
			".webp": "image/webp",
			".mp3":  "audio/mpeg",
			".wav":  "audio/wav",
			".flac": "audio/flac",
			".ogg":  "audio/ogg",
			".mp4":  "video/mp4",
			".avi":  "video/x-msvideo",
			".mkv":  "video/x-matroska",
			".webm": "video/webm",
			".mov":  "video/quicktime",
		},
	}
}

// DetectMimeType detects the MIME type of a file based on its extension
func (ftd *FileTypeDetector) DetectMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if mimeType, exists := ftd.mimeTypes[ext]; exists {
		return mimeType
	}
	return "application/octet-stream"
}

// IsImage checks if a file is an image based on its extension
func (ftd *FileTypeDetector) IsImage(path string) bool {
	mimeType := ftd.DetectMimeType(path)
	return strings.HasPrefix(mimeType, "image/")
}

// IsVideo checks if a file is a video based on its extension
func (ftd *FileTypeDetector) IsVideo(path string) bool {
	mimeType := ftd.DetectMimeType(path)
	return strings.HasPrefix(mimeType, "video/")
}

// IsAudio checks if a file is audio based on its extension
func (ftd *FileTypeDetector) IsAudio(path string) bool {
	mimeType := ftd.DetectMimeType(path)
	return strings.HasPrefix(mimeType, "audio/")
}

// IsText checks if a file is text based on its extension
func (ftd *FileTypeDetector) IsText(path string) bool {
	mimeType := ftd.DetectMimeType(path)
	return strings.HasPrefix(mimeType, "text/")
}

// GetCategory returns a general category for the file type
func (ftd *FileTypeDetector) GetCategory(path string) string {
	mimeType := ftd.DetectMimeType(path)

	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return "images"
	case strings.HasPrefix(mimeType, "video/"):
		return "videos"
	case strings.HasPrefix(mimeType, "audio/"):
		return "audio"
	case strings.HasPrefix(mimeType, "text/"):
		return "documents"
	case mimeType == "application/pdf":
		return "documents"
	case strings.Contains(mimeType, "zip") || strings.Contains(mimeType, "tar") || strings.Contains(mimeType, "gzip"):
		return "archives"
	default:
		return "other"
	}
}

// MockFileSystem provides a mock implementation for testing
type MockFileSystem struct {
	files map[string]*domain.FileInfo
	dirs  map[string]bool
}

// NewMockFileSystem creates a new mock file system for testing
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		files: make(map[string]*domain.FileInfo),
		dirs:  make(map[string]bool),
	}
}

// AddFile adds a file to the mock file system
func (mfs *MockFileSystem) AddFile(path string, size int64, modTime time.Time) {
	mfs.files[path] = &domain.FileInfo{
		Path:    path,
		Name:    filepath.Base(path),
		Size:    size,
		ModTime: modTime,
		IsDir:   false,
		Mode:    0644,
	}
}

// AddDir adds a directory to the mock file system
func (mfs *MockFileSystem) AddDir(path string) {
	mfs.dirs[path] = true
	mfs.files[path] = &domain.FileInfo{
		Path:    path,
		Name:    filepath.Base(path),
		Size:    0,
		ModTime: time.Now(),
		IsDir:   true,
		Mode:    0755,
	}
}

// Walk implements the FileSystem interface for testing
func (mfs *MockFileSystem) Walk(ctx context.Context, path string, fn domain.WalkFunc) error {
	for filePath, info := range mfs.files {
		if strings.HasPrefix(filePath, path) {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if err := fn(filePath, info, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

// Stat implements the FileSystem interface for testing
func (mfs *MockFileSystem) Stat(path string) (*domain.FileInfo, error) {
	if info, exists := mfs.files[path]; exists {
		return info, nil
	}
	return nil, os.ErrNotExist
}

// Remove implements the FileSystem interface for testing
func (mfs *MockFileSystem) Remove(path string) error {
	delete(mfs.files, path)
	delete(mfs.dirs, path)
	return nil
}

// RemoveAll implements the FileSystem interface for testing
func (mfs *MockFileSystem) RemoveAll(path string) error {
	for filePath := range mfs.files {
		if strings.HasPrefix(filePath, path) {
			delete(mfs.files, filePath)
		}
	}
	for dirPath := range mfs.dirs {
		if strings.HasPrefix(dirPath, path) {
			delete(mfs.dirs, dirPath)
		}
	}
	return nil
}

// Move implements the FileSystem interface for testing
func (mfs *MockFileSystem) Move(source, destination string) error {
	if info, exists := mfs.files[source]; exists {
		delete(mfs.files, source)
		info.Path = destination
		info.Name = filepath.Base(destination)
		mfs.files[destination] = info
	}
	return nil
}

// Copy implements the FileSystem interface for testing
func (mfs *MockFileSystem) Copy(source, destination string) error {
	if info, exists := mfs.files[source]; exists {
		newInfo := *info
		newInfo.Path = destination
		newInfo.Name = filepath.Base(destination)
		mfs.files[destination] = &newInfo
	}
	return nil
}

// CreateDir implements the FileSystem interface for testing
func (mfs *MockFileSystem) CreateDir(path string) error {
	mfs.AddDir(path)
	return nil
}

// IsEmpty implements the FileSystem interface for testing
func (mfs *MockFileSystem) IsEmpty(path string) (bool, error) {
	for filePath := range mfs.files {
		if strings.HasPrefix(filePath, path) && filePath != path {
			return false, nil
		}
	}
	return true, nil
}

// Exists implements the FileSystem interface for testing
func (mfs *MockFileSystem) Exists(path string) bool {
	_, exists := mfs.files[path]
	return exists
}

// ComputeHash implements the FileSystem interface for testing
func (mfs *MockFileSystem) ComputeHash(path string, algorithm string) (string, error) {
	if _, exists := mfs.files[path]; !exists {
		return "", os.ErrNotExist
	}
	// Return a mock hash for testing
	return fmt.Sprintf("mock_%s_%s", algorithm, filepath.Base(path)), nil
}
