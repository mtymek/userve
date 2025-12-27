# File Uploading

Capability for receiving files from remote clients via HTTP upload forms.

## ADDED Requirements

### Requirement: Upload Mode

The system SHALL provide an upload form allowing others to send files to the user.

#### Scenario: Start upload server
- **WHEN** user runs `userve -U`
- **THEN** the system starts an HTTP server with an upload form
- **AND** displays the URL where others can upload files

#### Scenario: File upload
- **WHEN** a client submits a file via the upload form
- **THEN** the file is saved to the current directory
- **AND** the system displays the filename and size received

#### Scenario: Upload count limit
- **WHEN** user runs `userve -U -c 3`
- **THEN** the system accepts up to 3 file uploads
- **AND** terminates after the 3rd upload completes
