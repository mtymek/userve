# Change: Add Single File Serving Capability (MVP)

## Why

Users frequently need to share files over a local network quickly without setting up complex infrastructure. Existing solutions require either both parties to have special software installed, or maintaining permanent services. uServe solves this by providing a minimal, ad-hoc HTTP server that serves a single file.

## What Changes

- Add core HTTP server that binds to a configurable port and IP address
- Implement single-file serving with automatic MIME type detection
- Display URL for recipient to download the file
- Show progress during downloads
- Support graceful shutdown via Ctrl+C

## Impact

- Affected specs: `file-serving` (new capability)
- Affected code: New Go application entry point and HTTP handler
- Dependencies: Go standard library only (net/http, mime)
