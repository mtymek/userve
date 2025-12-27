# Change: Add Directory Serving

## Why

The MVP provides basic single-file serving. Users need to share entire directories, which requires serving them as compressed archives in various formats.

## What Changes

- Add directory serving with multiple compression formats (tar.gz, zip, tar)
- Add configurable download count limit with auto-termination
- Stream archives directly to response (no temp files)

## Impact

- Affected specs: `file-serving` (modify existing capability)
- Affected code: Extend HTTP handler, add archive creation
- Dependencies: Go standard library (archive/tar, archive/zip, compress/gzip)
