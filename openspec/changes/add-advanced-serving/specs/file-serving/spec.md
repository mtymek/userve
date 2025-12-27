# File Serving - Phase 2

Advanced serving features building on the MVP.

## ADDED Requirements

### Requirement: Download Count Limit

The system SHALL limit the number of times a file can be downloaded and terminate after the limit is reached.

#### Scenario: Default single download
- **WHEN** user runs `userve <filepath>` without count option
- **THEN** the file can be downloaded exactly once
- **AND** the server terminates after the download completes

#### Scenario: Custom download count
- **WHEN** user runs `userve -c 5 <filepath>`
- **THEN** the file can be downloaded up to 5 times
- **AND** the server terminates after the 5th download completes

#### Scenario: Unlimited downloads
- **WHEN** user runs `userve -c 0 <filepath>`
- **THEN** the file can be downloaded unlimited times
- **AND** the server continues running until manually terminated

#### Scenario: Remaining count display
- **WHEN** a download completes and more downloads remain
- **THEN** the system displays how many downloads are remaining

### Requirement: Directory Serving

The system SHALL serve directories as compressed archives.

#### Scenario: Default directory compression
- **WHEN** user runs `userve <dirpath>` with a directory
- **THEN** the system creates a gzip-compressed tar archive of the directory
- **AND** serves it with filename `<dirname>.tar.gz`

#### Scenario: ZIP compression
- **WHEN** user runs `userve -Z <dirpath>` with a directory
- **THEN** the system creates a ZIP archive of the directory
- **AND** serves it with filename `<dirname>.zip`

#### Scenario: Bzip2 compression
- **WHEN** user runs `userve -j <dirpath>` with a directory
- **THEN** the system creates a bzip2-compressed tar archive
- **AND** serves it with filename `<dirname>.tar.bz2`

#### Scenario: No compression
- **WHEN** user runs `userve -u <dirpath>` with a directory
- **THEN** the system creates an uncompressed tar archive
- **AND** serves it with filename `<dirname>.tar`

#### Scenario: Gzip compression explicit
- **WHEN** user runs `userve -z <dirpath>` with a directory
- **THEN** the system creates a gzip-compressed tar archive (same as default)

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

### Requirement: Self Distribution

The system SHALL be able to serve its own binary for easy distribution.

#### Scenario: Serve self
- **WHEN** user runs `userve -s`
- **THEN** the system serves its own executable binary
- **AND** displays the URL where others can download userve
