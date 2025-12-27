package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"flag"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const defaultPort = 8080

// ArchiveFormat represents the archive format for directories
type ArchiveFormat int

const (
	ArchiveTarGz ArchiveFormat = iota
	ArchiveZip
	ArchiveTar
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("userve", flag.ContinueOnError)
	port := fs.Int("p", defaultPort, "port to listen on")
	bindIP := fs.String("i", "", "IP address to bind to (default: all interfaces)")
	count := fs.Int("c", 1, "number of downloads allowed (0 for unlimited)")
	archiveFormat := fs.String("a", "tar.gz", "archive format for directories: tar.gz, zip, tar")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: userve [options] <file|directory>\n\n")
		fmt.Fprintf(os.Stderr, "Serve a file or directory over HTTP on your local network.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		fs.Usage()
		return fmt.Errorf("file path required")
	}

	filePath := fs.Arg(0)

	// Parse archive format
	var format ArchiveFormat
	switch *archiveFormat {
	case "tar.gz":
		format = ArchiveTarGz
	case "zip":
		format = ArchiveZip
	case "tar":
		format = ArchiveTar
	default:
		return fmt.Errorf("invalid archive format %q: valid formats are tar.gz, zip, tar", *archiveFormat)
	}

	// Validate file exists
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", filePath)
	}
	if err != nil {
		return fmt.Errorf("cannot access file: %v", err)
	}

	// Determine bind address
	bindAddr := "0.0.0.0"
	if *bindIP != "" {
		bindAddr = *bindIP
	}

	// Determine display IP (for URL)
	displayIP := *bindIP
	if displayIP == "" {
		displayIP = getLocalIP()
	}

	addr := fmt.Sprintf("%s:%d", bindAddr, *port)

	// Create listener first to detect port-in-use errors early
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("cannot bind to %s: %v", addr, err)
	}

	// Track active downloads for graceful shutdown
	var activeDownloads sync.WaitGroup

	// Channel to signal when download limit reached
	downloadComplete := make(chan struct{}, 1)

	// Create appropriate content provider
	var provider contentProvider
	if info.IsDir() {
		provider = &archiveProvider{
			dirPath: filePath,
			dirName: filepath.Base(filePath),
			format:  format,
		}
	} else {
		provider = &fileProvider{
			filePath: filePath,
			fileName: filepath.Base(filePath),
			fileSize: info.Size(),
		}
	}

	h := &handler{
		provider:         provider,
		activeDownloads:  &activeDownloads,
		downloadComplete: downloadComplete,
		maxDownloads:     int32(*count),
	}
	displayName := provider.Filename()

	server := &http.Server{
		Handler: h,
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Serve(listener)
	}()

	url := fmt.Sprintf("http://%s:%d/%s", displayIP, *port, displayName)
	fmt.Printf("Serving %s\n", filePath)
	fmt.Printf("URL: %s\n", url)
	if *count == 0 {
		fmt.Printf("Downloads: unlimited\n")
	} else {
		fmt.Printf("Downloads: %d remaining\n", *count)
	}
	fmt.Printf("Press Ctrl+C to stop\n")

	select {
	case sig := <-sigChan:
		fmt.Printf("\nReceived %v, shutting down...\n", sig)
	case err := <-errChan:
		if err != http.ErrServerClosed {
			return fmt.Errorf("server error: %v", err)
		}
	case <-downloadComplete:
		fmt.Println("Download limit reached, shutting down...")
	}

	// Graceful shutdown: stop accepting new connections
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown stops accepting new connections but doesn't wait for handlers
	server.Shutdown(ctx)

	// Wait for active downloads to complete
	done := make(chan struct{})
	go func() {
		activeDownloads.Wait()
		close(done)
	}()

	select {
	case <-done:
		fmt.Println("All downloads completed")
	case <-ctx.Done():
		fmt.Println("Shutdown timeout reached")
	}

	return nil
}

// contentProvider abstracts the content being served (file or archive)
type contentProvider interface {
	// Filename returns the name to use in Content-Disposition
	Filename() string
	// ContentType returns the MIME type
	ContentType() string
	// ContentLength returns the size if known, or -1 for streaming
	ContentLength() int64
	// WriteTo writes the content to the writer
	WriteTo(w io.Writer) error
}

// handler is the unified HTTP handler for serving any content
type handler struct {
	provider         contentProvider
	activeDownloads  *sync.WaitGroup
	downloadComplete chan struct{}
	maxDownloads     int32
	downloadCount    atomic.Int32
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.activeDownloads.Add(1)
	defer h.activeDownloads.Done()

	remoteAddr := r.RemoteAddr
	fmt.Printf("[%s] Download started from %s\n", time.Now().Format("15:04:05"), remoteAddr)

	// Set headers
	w.Header().Set("Content-Type", h.provider.ContentType())
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", h.provider.Filename()))
	if length := h.provider.ContentLength(); length >= 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", length))
	}

	// Serve content
	if err := h.provider.WriteTo(w); err != nil {
		fmt.Printf("[%s] Download interrupted from %s: %v\n", time.Now().Format("15:04:05"), remoteAddr, err)
		return
	}

	fmt.Printf("[%s] Download completed from %s\n", time.Now().Format("15:04:05"), remoteAddr)

	// Track download count
	newCount := h.downloadCount.Add(1)

	// Check if we've reached the limit
	if h.maxDownloads == 0 {
		// Unlimited mode - no shutdown
		return
	}

	remaining := h.maxDownloads - newCount
	if remaining > 0 {
		fmt.Printf("[%s] %d download(s) remaining\n", time.Now().Format("15:04:05"), remaining)
	} else {
		// Signal shutdown when limit reached
		select {
		case h.downloadComplete <- struct{}{}:
		default:
		}
	}
}

// fileProvider serves a single file
type fileProvider struct {
	filePath string
	fileName string
	fileSize int64
}

func (p *fileProvider) Filename() string {
	return p.fileName
}

func (p *fileProvider) ContentType() string {
	contentType := mime.TypeByExtension(filepath.Ext(p.fileName))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return contentType
}

func (p *fileProvider) ContentLength() int64 {
	return p.fileSize
}

func (p *fileProvider) WriteTo(w io.Writer) error {
	file, err := os.Open(p.filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(w, file)
	return err
}

// archiveProvider serves a directory as an archive
type archiveProvider struct {
	dirPath string
	dirName string
	format  ArchiveFormat
}

func (p *archiveProvider) Filename() string {
	switch p.format {
	case ArchiveTarGz:
		return p.dirName + ".tar.gz"
	case ArchiveZip:
		return p.dirName + ".zip"
	case ArchiveTar:
		return p.dirName + ".tar"
	default:
		return p.dirName + ".tar.gz"
	}
}

func (p *archiveProvider) ContentType() string {
	switch p.format {
	case ArchiveTarGz:
		return "application/gzip"
	case ArchiveZip:
		return "application/zip"
	case ArchiveTar:
		return "application/x-tar"
	default:
		return "application/gzip"
	}
}

func (p *archiveProvider) ContentLength() int64 {
	return -1 // Streaming, unknown size
}

func (p *archiveProvider) WriteTo(w io.Writer) error {
	if p.format == ArchiveZip {
		return p.writeZipArchive(w)
	}
	return p.writeTarArchive(w)
}

func (p *archiveProvider) writeTarArchive(w io.Writer) error {
	var tw *tar.Writer

	switch p.format {
	case ArchiveTarGz:
		gw := gzip.NewWriter(w)
		defer gw.Close()
		tw = tar.NewWriter(gw)
	case ArchiveTar:
		tw = tar.NewWriter(w)
	default:
		gw := gzip.NewWriter(w)
		defer gw.Close()
		tw = tar.NewWriter(gw)
	}
	defer tw.Close()

	baseDir := filepath.Base(p.dirPath)

	return filepath.Walk(p.dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		// Adjust the name to be relative to the directory being archived
		relPath, err := filepath.Rel(p.dirPath, path)
		if err != nil {
			return err
		}
		header.Name = filepath.Join(baseDir, relPath)

		// Write header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// If it's a file, write its contents
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(tw, file); err != nil {
				return err
			}
		}

		return nil
	})
}

func (p *archiveProvider) writeZipArchive(w io.Writer) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	baseDir := filepath.Base(p.dirPath)

	return filepath.Walk(p.dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create zip header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Adjust the name to be relative to the directory being archived
		relPath, err := filepath.Rel(p.dirPath, path)
		if err != nil {
			return err
		}
		header.Name = filepath.Join(baseDir, relPath)

		// Ensure directories end with /
		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		// Write header
		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}

		// If it's a file, write its contents
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(writer, file); err != nil {
				return err
			}
		}

		return nil
	})
}

func getLocalIP() string {
	// Try to get the preferred outbound IP
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
