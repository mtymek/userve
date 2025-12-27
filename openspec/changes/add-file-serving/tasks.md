# Implementation Tasks

## 1. Project Setup
- [ ] 1.1 Initialize Go module (`go mod init`)
- [ ] 1.2 Create main.go entry point
- [ ] 1.3 Set up basic project structure

## 2. CLI Argument Parsing
- [ ] 2.1 Implement flag parsing for `-p` (port)
- [ ] 2.2 Implement flag parsing for `-i` (IP address)
- [ ] 2.3 Validate file path argument exists and is not a directory
- [ ] 2.4 Display usage/help message

## 3. Network Setup
- [ ] 3.1 Detect primary local IP address for URL display
- [ ] 3.2 Create HTTP server with configurable bind address
- [ ] 3.3 Handle port-in-use errors gracefully

## 4. File Serving
- [ ] 4.1 Implement file handler that serves the specified file
- [ ] 4.2 Detect MIME type from file extension
- [ ] 4.3 Set Content-Disposition header with original filename
- [ ] 4.4 Set Content-Length header

## 5. Progress & Output
- [ ] 5.1 Display URL on startup
- [ ] 5.2 Log incoming requests with remote address
- [ ] 5.3 Display download completion message

## 6. Graceful Shutdown
- [ ] 6.1 Handle SIGINT/SIGTERM signals
- [ ] 6.2 Complete in-progress downloads before exit
- [ ] 6.3 Clean exit with appropriate status code

## 7. Testing
- [ ] 7.1 Unit tests for MIME type detection
- [ ] 7.2 Unit tests for CLI argument parsing
- [ ] 7.3 Integration tests for file serving
