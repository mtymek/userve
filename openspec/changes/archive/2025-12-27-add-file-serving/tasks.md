# Implementation Tasks

## 1. Project Setup
- [x] 1.1 Initialize Go module (`go mod init`)
- [x] 1.2 Create main.go entry point
- [x] 1.3 Set up basic project structure

## 2. CLI Argument Parsing
- [x] 2.1 Implement flag parsing for `-p` (port)
- [x] 2.2 Implement flag parsing for `-i` (IP address)
- [x] 2.3 Validate file path argument exists and is not a directory
- [x] 2.4 Display usage/help message

## 3. Network Setup
- [x] 3.1 Detect primary local IP address for URL display
- [x] 3.2 Create HTTP server with configurable bind address
- [x] 3.3 Handle port-in-use errors gracefully

## 4. File Serving
- [x] 4.1 Implement file handler that serves the specified file
- [x] 4.2 Detect MIME type from file extension
- [x] 4.3 Set Content-Disposition header with original filename
- [x] 4.4 Set Content-Length header

## 5. Progress & Output
- [x] 5.1 Display URL on startup
- [x] 5.2 Log incoming requests with remote address
- [x] 5.3 Display download completion message

## 6. One-Shot Mode & Graceful Shutdown
- [x] 6.1 Auto-exit after first successful download (one-shot mode)
- [x] 6.2 Handle SIGINT/SIGTERM signals
- [x] 6.3 Complete in-progress downloads before exit
- [x] 6.4 Clean exit with appropriate status code

## 7. Testing
- [x] 7.1 Unit tests for MIME type detection
- [x] 7.2 Unit tests for CLI argument parsing
- [x] 7.3 Integration tests for file serving
