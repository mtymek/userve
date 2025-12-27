# File Serving

Core capability for serving a single file over HTTP on a local network.

## ADDED Requirements

### Requirement: Single File Serving

The system SHALL serve a specified file via HTTP when invoked with a file path argument.

#### Scenario: Serve existing file
- **WHEN** user runs `userve <filepath>`
- **THEN** the system starts an HTTP server
- **AND** displays the URL where the file can be downloaded
- **AND** the file is available for download at that URL
- **AND** the system exits automatically after the file is downloaded once

#### Scenario: File not found
- **WHEN** user runs `userve <filepath>` with a non-existent file
- **THEN** the system exits with an error message indicating the file was not found

#### Scenario: Path is a directory
- **WHEN** user runs `userve <dirpath>` with a directory path
- **THEN** the system exits with an error message indicating directories are not supported

### Requirement: Network Binding

The system SHALL bind to a configurable IP address and port for serving files.

#### Scenario: Default binding
- **WHEN** user runs `userve <filepath>` without network options
- **THEN** the system binds to all interfaces (0.0.0.0) on port 8080
- **AND** displays the URL using the machine's primary local IP address

#### Scenario: Custom port
- **WHEN** user runs `userve -p 9000 <filepath>`
- **THEN** the system binds to port 9000

#### Scenario: Custom IP address
- **WHEN** user runs `userve -i 192.168.1.100 <filepath>`
- **THEN** the system binds only to the specified IP address
- **AND** displays the URL using that IP address

#### Scenario: Port in use
- **WHEN** user runs `userve -p <port>` and the port is already in use
- **THEN** the system exits with an error message indicating the port is unavailable

### Requirement: MIME Type Detection

The system SHALL detect and set appropriate Content-Type headers based on file extension.

#### Scenario: Known file type
- **WHEN** a client requests a file with a recognized extension (e.g., .pdf, .jpg, .txt)
- **THEN** the response includes the appropriate Content-Type header

#### Scenario: Unknown file type
- **WHEN** a client requests a file with an unrecognized extension
- **THEN** the response uses Content-Type: application/octet-stream

### Requirement: Download Headers

The system SHALL include headers that facilitate file download in browsers.

#### Scenario: Content-Disposition header
- **WHEN** a client downloads a file
- **THEN** the response includes Content-Disposition header with the original filename
- **AND** the response includes Content-Length header with the file size

### Requirement: Progress Display

The system SHALL display progress information during file transfers.

#### Scenario: Download started
- **WHEN** a client connects and begins downloading
- **THEN** the system displays the remote address

#### Scenario: Download completed
- **WHEN** a download completes successfully
- **THEN** the system displays a completion message

### Requirement: One-Shot Download

The system SHALL exit automatically after a single successful download by default.

#### Scenario: Auto-exit after download
- **WHEN** a client successfully completes downloading the file
- **THEN** the system displays a completion message
- **AND** exits cleanly with status code 0

### Requirement: Graceful Shutdown

The system SHALL support graceful shutdown via interrupt signals.

#### Scenario: Interrupt signal
- **WHEN** user sends SIGINT (Ctrl+C) while the server is running
- **THEN** the system stops accepting new connections
- **AND** completes any in-progress downloads
- **AND** exits cleanly
