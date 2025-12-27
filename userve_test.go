package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestRunFileNotFound(t *testing.T) {
	err := run([]string{"/nonexistent/path/to/file.txt"})
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("expected 'file not found' error, got: %v", err)
	}
}

func TestRunMissingFilePath(t *testing.T) {
	err := run([]string{})
	if err == nil {
		t.Fatal("expected error for missing file path")
	}
	if !strings.Contains(err.Error(), "file path required") {
		t.Errorf("expected 'file path required' error, got: %v", err)
	}
}

func TestRunInvalidFlag(t *testing.T) {
	err := run([]string{"--invalid-flag"})
	if err == nil {
		t.Fatal("expected error for invalid flag")
	}
}

func TestRunInvalidArchiveFormat(t *testing.T) {
	tmpDir := t.TempDir()
	err := run([]string{"-a", "invalid", tmpDir})
	if err == nil {
		t.Fatal("expected error for invalid archive format")
	}
	if !strings.Contains(err.Error(), "invalid archive format") {
		t.Errorf("expected 'invalid archive format' error, got: %v", err)
	}
}

func TestFileHandler(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "testfile.txt")
	testContent := "Hello, World!"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	info, _ := os.Stat(testFile)
	var wg sync.WaitGroup
	downloadComplete := make(chan struct{}, 1)

	h := &handler{
		provider: &fileProvider{
			filePath: testFile,
			fileName: "testfile.txt",
			fileSize: info.Size(),
		},
		activeDownloads:  &wg,
		downloadComplete: downloadComplete,
		maxDownloads:     1,
	}

	req := httptest.NewRequest("GET", "/testfile.txt", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Check Content-Type
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/plain") {
		t.Errorf("expected Content-Type text/plain, got %q", contentType)
	}

	// Check Content-Disposition
	contentDisposition := resp.Header.Get("Content-Disposition")
	if !strings.Contains(contentDisposition, "testfile.txt") {
		t.Errorf("expected Content-Disposition to contain filename, got %q", contentDisposition)
	}

	// Check Content-Length
	contentLength := resp.Header.Get("Content-Length")
	if contentLength != "13" {
		t.Errorf("expected Content-Length 13, got %q", contentLength)
	}

	// Check body
	body, _ := io.ReadAll(resp.Body)
	if string(body) != testContent {
		t.Errorf("expected body %q, got %q", testContent, string(body))
	}
}

func TestGetLocalIP(t *testing.T) {
	ip := getLocalIP()
	if ip == "" {
		t.Error("getLocalIP returned empty string")
	}
	// Should return either a valid IP or fallback to localhost
	if ip != "127.0.0.1" && !strings.Contains(ip, ".") {
		t.Errorf("getLocalIP returned invalid IP: %s", ip)
	}
}

func TestFileHandlerSignalsDownloadComplete(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "testfile.txt")
	testContent := "Hello, World!"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	info, _ := os.Stat(testFile)
	var wg sync.WaitGroup
	downloadComplete := make(chan struct{}, 1)

	h := &handler{
		provider: &fileProvider{
			filePath: testFile,
			fileName: "testfile.txt",
			fileSize: info.Size(),
		},
		activeDownloads:  &wg,
		downloadComplete: downloadComplete,
		maxDownloads:     1,
	}

	req := httptest.NewRequest("GET", "/testfile.txt", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	// Check that downloadComplete was signaled
	select {
	case <-downloadComplete:
		// Success - signal was sent
	default:
		t.Error("expected downloadComplete to be signaled after successful download")
	}
}

func TestFileHandlerDownloadCounting(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "testfile.txt")
	testContent := "Hello, World!"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	info, _ := os.Stat(testFile)
	var wg sync.WaitGroup
	downloadComplete := make(chan struct{}, 1)

	h := &handler{
		provider: &fileProvider{
			filePath: testFile,
			fileName: "testfile.txt",
			fileSize: info.Size(),
		},
		activeDownloads:  &wg,
		downloadComplete: downloadComplete,
		maxDownloads:     3, // Allow 3 downloads
	}

	// First two downloads should not signal completion
	for i := range 2 {
		req := httptest.NewRequest("GET", "/testfile.txt", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		select {
		case <-downloadComplete:
			t.Errorf("downloadComplete signaled too early on download %d", i+1)
		default:
			// Expected - not complete yet
		}
	}

	// Third download should signal completion
	req := httptest.NewRequest("GET", "/testfile.txt", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	select {
	case <-downloadComplete:
		// Success - signal was sent after 3rd download
	default:
		t.Error("expected downloadComplete to be signaled after 3rd download")
	}
}

func TestFileHandlerUnlimitedDownloads(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "testfile.txt")
	testContent := "Hello, World!"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	info, _ := os.Stat(testFile)
	var wg sync.WaitGroup
	downloadComplete := make(chan struct{}, 1)

	h := &handler{
		provider: &fileProvider{
			filePath: testFile,
			fileName: "testfile.txt",
			fileSize: info.Size(),
		},
		activeDownloads:  &wg,
		downloadComplete: downloadComplete,
		maxDownloads:     0, // Unlimited
	}

	// Multiple downloads should not signal completion
	for i := range 5 {
		req := httptest.NewRequest("GET", "/testfile.txt", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		select {
		case <-downloadComplete:
			t.Errorf("downloadComplete signaled unexpectedly on download %d", i+1)
		default:
			// Expected - unlimited mode never signals
		}
	}
}

func TestArchiveProviderFilename(t *testing.T) {
	tests := []struct {
		format   ArchiveFormat
		expected string
	}{
		{ArchiveTarGz, "testdir.tar.gz"},
		{ArchiveZip, "testdir.zip"},
		{ArchiveTar, "testdir.tar"},
	}

	for _, tt := range tests {
		p := &archiveProvider{
			dirName: "testdir",
			format:  tt.format,
		}
		result := p.Filename()
		if result != tt.expected {
			t.Errorf("Filename() with format %v = %q, want %q", tt.format, result, tt.expected)
		}
	}
}

func TestArchiveProviderContentType(t *testing.T) {
	tests := []struct {
		format   ArchiveFormat
		expected string
	}{
		{ArchiveTarGz, "application/gzip"},
		{ArchiveZip, "application/zip"},
		{ArchiveTar, "application/x-tar"},
	}

	for _, tt := range tests {
		p := &archiveProvider{
			dirName: "testdir",
			format:  tt.format,
		}
		result := p.ContentType()
		if result != tt.expected {
			t.Errorf("ContentType() with format %v = %q, want %q", tt.format, result, tt.expected)
		}
	}
}

func TestDirHandlerTarGzArchive(t *testing.T) {
	// Create a temporary directory with files
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "testdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("content1"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	subDir := filepath.Join(testDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("content2"), 0644); err != nil {
		t.Fatalf("failed to create test file in subdir: %v", err)
	}

	var wg sync.WaitGroup
	downloadComplete := make(chan struct{}, 1)

	h := &handler{
		provider: &archiveProvider{
			dirPath: testDir,
			dirName: "testdir",
			format:  ArchiveTarGz,
		},
		activeDownloads:  &wg,
		downloadComplete: downloadComplete,
		maxDownloads:     1,
	}

	req := httptest.NewRequest("GET", "/testdir.tar.gz", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Check headers
	if ct := resp.Header.Get("Content-Type"); ct != "application/gzip" {
		t.Errorf("expected Content-Type application/gzip, got %q", ct)
	}
	if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, "testdir.tar.gz") {
		t.Errorf("expected Content-Disposition to contain testdir.tar.gz, got %q", cd)
	}

	// Verify archive contents
	body, _ := io.ReadAll(resp.Body)
	gr, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	files := make(map[string]bool)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("failed to read tar entry: %v", err)
		}
		files[header.Name] = true
	}

	expectedFiles := []string{"testdir", "testdir/file1.txt", "testdir/subdir", "testdir/subdir/file2.txt"}
	for _, f := range expectedFiles {
		if !files[f] {
			t.Errorf("expected file %q in archive, got files: %v", f, files)
		}
	}
}

func TestDirHandlerZipArchive(t *testing.T) {
	// Create a temporary directory with files
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "testdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("content1"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	var wg sync.WaitGroup
	downloadComplete := make(chan struct{}, 1)

	h := &handler{
		provider: &archiveProvider{
			dirPath: testDir,
			dirName: "testdir",
			format:  ArchiveZip,
		},
		activeDownloads:  &wg,
		downloadComplete: downloadComplete,
		maxDownloads:     1,
	}

	req := httptest.NewRequest("GET", "/testdir.zip", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Check headers
	if ct := resp.Header.Get("Content-Type"); ct != "application/zip" {
		t.Errorf("expected Content-Type application/zip, got %q", ct)
	}

	// Verify archive contents
	body, _ := io.ReadAll(resp.Body)
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("failed to create zip reader: %v", err)
	}

	files := make(map[string]bool)
	for _, f := range zr.File {
		files[f.Name] = true
	}

	expectedFiles := []string{"testdir/", "testdir/file1.txt"}
	for _, f := range expectedFiles {
		if !files[f] {
			t.Errorf("expected file %q in archive, got files: %v", f, files)
		}
	}
}

func TestDirHandlerUncompressedTar(t *testing.T) {
	// Create a temporary directory with a file
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "testdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("content1"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	var wg sync.WaitGroup
	downloadComplete := make(chan struct{}, 1)

	h := &handler{
		provider: &archiveProvider{
			dirPath: testDir,
			dirName: "testdir",
			format:  ArchiveTar,
		},
		activeDownloads:  &wg,
		downloadComplete: downloadComplete,
		maxDownloads:     1,
	}

	req := httptest.NewRequest("GET", "/testdir.tar", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Check headers
	if ct := resp.Header.Get("Content-Type"); ct != "application/x-tar" {
		t.Errorf("expected Content-Type application/x-tar, got %q", ct)
	}

	// Verify archive contents (uncompressed tar)
	body, _ := io.ReadAll(resp.Body)
	tr := tar.NewReader(bytes.NewReader(body))
	files := make(map[string]bool)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("failed to read tar entry: %v", err)
		}
		files[header.Name] = true
	}

	expectedFiles := []string{"testdir", "testdir/file1.txt"}
	for _, f := range expectedFiles {
		if !files[f] {
			t.Errorf("expected file %q in archive, got files: %v", f, files)
		}
	}
}

func TestDirHandlerDownloadCounting(t *testing.T) {
	// Create a temporary directory with a file
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "testdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("content1"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	var wg sync.WaitGroup
	downloadComplete := make(chan struct{}, 1)

	h := &handler{
		provider: &archiveProvider{
			dirPath: testDir,
			dirName: "testdir",
			format:  ArchiveTarGz,
		},
		activeDownloads:  &wg,
		downloadComplete: downloadComplete,
		maxDownloads:     2,
	}

	// First download should not signal completion
	req := httptest.NewRequest("GET", "/testdir.tar.gz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	select {
	case <-downloadComplete:
		t.Error("downloadComplete signaled too early on first download")
	default:
		// Expected
	}

	// Second download should signal completion
	req = httptest.NewRequest("GET", "/testdir.tar.gz", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	select {
	case <-downloadComplete:
		// Success
	default:
		t.Error("expected downloadComplete to be signaled after 2nd download")
	}
}
