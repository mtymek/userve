# Change: Add Advanced Serving Features (Phase 2)

## Why

The MVP provides basic single-file serving. Users need additional capabilities for real-world use: limiting downloads to prevent over-sharing, serving entire directories as archives, and receiving files from others via upload forms.

## What Changes

- Add configurable download count limit with auto-termination
- Add directory serving with multiple compression formats (tar.gz, zip, bz2, uncompressed)
- Add upload mode for receiving files from others
- Add self-distribution mode (serve the userve binary itself)

## Impact

- Affected specs: `file-serving` (modify existing capability), `file-uploading` (new capability)
- Affected code: Extend HTTP handler, add archive creation, add upload handler
- Dependencies: Go standard library (archive/tar, archive/zip, compress/gzip, compress/bzip2)
