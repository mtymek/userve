package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestDetectMIMEType(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"document.pdf", "application/pdf"},
		{"image.jpg", "image/jpeg"},
		{"image.jpeg", "image/jpeg"},
		{"image.png", "image/png"},
		{"file.txt", "text/plain; charset=utf-8"},
		{"page.html", "text/html; charset=utf-8"},
		{"data.json", "application/json"},
		{"archive.zip", "application/zip"},
		{"unknown.qzx", "application/octet-stream"},
		{"noextension", "application/octet-stream"},
		{"", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := detectMIMEType(tt.filename)
			if result != tt.expected {
				t.Errorf("detectMIMEType(%q) = %q, want %q", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestRunFileNotFound(t *testing.T) {
	err := run([]string{"/nonexistent/path/to/file.txt"})
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("expected 'file not found' error, got: %v", err)
	}
}

func TestRunDirectoryNotSupported(t *testing.T) {
	tmpDir := t.TempDir()

	err := run([]string{tmpDir})
	if err == nil {
		t.Fatal("expected error for directory")
	}
	if !strings.Contains(err.Error(), "directories are not supported") {
		t.Errorf("expected 'directories are not supported' error, got: %v", err)
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

	handler := &fileHandler{
		filePath:         testFile,
		fileName:         "testfile.txt",
		fileSize:         info.Size(),
		activeDownloads:  &wg,
		downloadComplete: downloadComplete,
	}

	req := httptest.NewRequest("GET", "/testfile.txt", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

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

	handler := &fileHandler{
		filePath:         testFile,
		fileName:         "testfile.txt",
		fileSize:         info.Size(),
		activeDownloads:  &wg,
		downloadComplete: downloadComplete,
	}

	req := httptest.NewRequest("GET", "/testfile.txt", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Check that downloadComplete was signaled
	select {
	case <-downloadComplete:
		// Success - signal was sent
	default:
		t.Error("expected downloadComplete to be signaled after successful download")
	}
}
