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

// validArchiveFormats lists accepted values for the -a flag
var validArchiveFormats = []string{"tar.gz", "zip", "tar"}

// parseArchiveFormat converts a string to ArchiveFormat
func parseArchiveFormat(s string) (ArchiveFormat, error) {
	switch s {
	case "tar.gz":
		return ArchiveTarGz, nil
	case "zip":
		return ArchiveZip, nil
	case "tar":
		return ArchiveTar, nil
	default:
		return ArchiveTarGz, fmt.Errorf("invalid archive format %q: valid formats are %v", s, validArchiveFormats)
	}
}

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
	format, err := parseArchiveFormat(*archiveFormat)
	if err != nil {
		return err
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

	// Create appropriate handler
	var handler http.Handler
	var displayName string

	if info.IsDir() {
		// Directory serving
		dirHandler := &dirHandler{
			dirPath:          filePath,
			dirName:          filepath.Base(filePath),
			format:           format,
			activeDownloads:  &activeDownloads,
			downloadComplete: downloadComplete,
			maxDownloads:     int32(*count),
		}
		handler = dirHandler
		displayName = dirHandler.archiveFilename()
	} else {
		// File serving
		fileHandler := &fileHandler{
			filePath:         filePath,
			fileName:         filepath.Base(filePath),
			fileSize:         info.Size(),
			activeDownloads:  &activeDownloads,
			downloadComplete: downloadComplete,
			maxDownloads:     int32(*count),
		}
		handler = fileHandler
		displayName = fileHandler.fileName
	}

	server := &http.Server{
		Handler: handler,
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

	// Wait for signal, server error, or download limit reached
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

type fileHandler struct {
	filePath         string
	fileName         string
	fileSize         int64
	activeDownloads  *sync.WaitGroup
	downloadComplete chan struct{}
	maxDownloads     int32
	downloadCount    atomic.Int32
}

func (h *fileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.activeDownloads.Add(1)
	defer h.activeDownloads.Done()

	remoteAddr := r.RemoteAddr
	fmt.Printf("[%s] Download started from %s\n", time.Now().Format("15:04:05"), remoteAddr)

	// Open file
	file, err := os.Open(h.filePath)
	if err != nil {
		fmt.Printf("[%s] Error opening file: %v\n", time.Now().Format("15:04:05"), err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Set headers
	contentType := detectMIMEType(h.fileName)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", h.fileName))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", h.fileSize))

	// Serve file
	written, err := io.Copy(w, file)
	if err != nil {
		fmt.Printf("[%s] Download interrupted from %s: %v\n", time.Now().Format("15:04:05"), remoteAddr, err)
		return
	}

	fmt.Printf("[%s] Download completed from %s (%d bytes)\n", time.Now().Format("15:04:05"), remoteAddr, written)

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

type dirHandler struct {
	dirPath          string
	dirName          string
	format           ArchiveFormat
	activeDownloads  *sync.WaitGroup
	downloadComplete chan struct{}
	maxDownloads     int32
	downloadCount    atomic.Int32
}

func (h *dirHandler) archiveFilename() string {
	switch h.format {
	case ArchiveTarGz:
		return h.dirName + ".tar.gz"
	case ArchiveZip:
		return h.dirName + ".zip"
	case ArchiveTar:
		return h.dirName + ".tar"
	default:
		return h.dirName + ".tar.gz"
	}
}

func (h *dirHandler) contentType() string {
	switch h.format {
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

func (h *dirHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.activeDownloads.Add(1)
	defer h.activeDownloads.Done()

	remoteAddr := r.RemoteAddr
	fmt.Printf("[%s] Download started from %s\n", time.Now().Format("15:04:05"), remoteAddr)

	// Set headers
	filename := h.archiveFilename()
	w.Header().Set("Content-Type", h.contentType())
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	// Note: Content-Length is not set for streaming archives

	var err error
	if h.format == ArchiveZip {
		err = h.writeZipArchive(w)
	} else {
		err = h.writeTarArchive(w)
	}

	if err != nil {
		fmt.Printf("[%s] Archive creation interrupted from %s: %v\n", time.Now().Format("15:04:05"), remoteAddr, err)
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

func (h *dirHandler) writeTarArchive(w io.Writer) error {
	var tw *tar.Writer

	switch h.format {
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

	baseDir := filepath.Base(h.dirPath)

	return filepath.Walk(h.dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		// Adjust the name to be relative to the directory being archived
		relPath, err := filepath.Rel(h.dirPath, path)
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

func (h *dirHandler) writeZipArchive(w io.Writer) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	baseDir := filepath.Base(h.dirPath)

	return filepath.Walk(h.dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create zip header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Adjust the name to be relative to the directory being archived
		relPath, err := filepath.Rel(h.dirPath, path)
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

func detectMIMEType(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return "application/octet-stream"
	}

	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		return "application/octet-stream"
	}

	return mimeType
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
