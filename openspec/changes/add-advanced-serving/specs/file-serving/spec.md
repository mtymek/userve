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

#### Scenario: Explicit tar.gz format
- **WHEN** user runs `userve -a tar.gz <dirpath>` with a directory
- **THEN** the system creates a gzip-compressed tar archive (same as default)
- **AND** serves it with filename `<dirname>.tar.gz`

#### Scenario: ZIP format
- **WHEN** user runs `userve -a zip <dirpath>` with a directory
- **THEN** the system creates a ZIP archive of the directory
- **AND** serves it with filename `<dirname>.zip`

#### Scenario: Uncompressed tar format
- **WHEN** user runs `userve -a tar <dirpath>` with a directory
- **THEN** the system creates an uncompressed tar archive
- **AND** serves it with filename `<dirname>.tar`

#### Scenario: Invalid archive format
- **WHEN** user runs `userve -a invalid <dirpath>` with an unrecognized format
- **THEN** the system exits with an error listing valid formats
