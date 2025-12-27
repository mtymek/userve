package main

import (
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
	"syscall"
	"time"
)

const defaultPort = 8080

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

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: userve [options] <file>\n\n")
		fmt.Fprintf(os.Stderr, "Serve a single file over HTTP on your local network.\n\n")
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

	// Validate file exists and is not a directory
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", filePath)
	}
	if err != nil {
		return fmt.Errorf("cannot access file: %v", err)
	}
	if info.IsDir() {
		return fmt.Errorf("directories are not supported: %s", filePath)
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

	// Channel to signal successful download completion (one-shot mode)
	downloadComplete := make(chan struct{}, 1)

	// Create file handler
	handler := &fileHandler{
		filePath:         filePath,
		fileName:         filepath.Base(filePath),
		fileSize:         info.Size(),
		activeDownloads:  &activeDownloads,
		downloadComplete: downloadComplete,
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

	url := fmt.Sprintf("http://%s:%d/%s", displayIP, *port, handler.fileName)
	fmt.Printf("Serving %s\n", filePath)
	fmt.Printf("URL: %s\n", url)
	fmt.Printf("Press Ctrl+C to stop\n")

	// Wait for signal, server error, or successful download
	select {
	case sig := <-sigChan:
		fmt.Printf("\nReceived %v, shutting down...\n", sig)
	case err := <-errChan:
		if err != http.ErrServerClosed {
			return fmt.Errorf("server error: %v", err)
		}
	case <-downloadComplete:
		fmt.Println("Download complete, shutting down...")
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

	// Signal successful download for one-shot mode
	select {
	case h.downloadComplete <- struct{}{}:
	default:
	}
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
