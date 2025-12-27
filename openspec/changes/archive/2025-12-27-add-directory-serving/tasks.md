# Implementation Tasks

## 1. Download Count Limiting
- [x] 1.1 Add `-c` flag for download count (default: 1)
- [x] 1.2 Track download count with atomic counter
- [x] 1.3 Decrement counter on successful download
- [x] 1.4 Shutdown server when count reaches zero
- [x] 1.5 Handle unlimited mode (count=0)
- [x] 1.6 Display remaining download count after each download

## 2. Directory Serving
- [x] 2.1 Detect if path is a directory
- [x] 2.2 Add `-a` flag for archive format (tar.gz, zip, tar)
- [x] 2.3 Implement tar.gz archive creation (default)
- [x] 2.4 Implement ZIP archive creation (`-a zip`)
- [x] 2.5 Implement uncompressed tar creation (`-a tar`)
- [x] 2.6 Stream archive directly to response (avoid temp files)
- [x] 2.7 Set appropriate filename in Content-Disposition
- [x] 2.8 Add error handling for invalid archive format

## 3. Testing
- [x] 3.1 Unit tests for download counting
- [x] 3.2 Integration tests for directory archiving (all formats)
