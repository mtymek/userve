# Implementation Tasks

## 1. Download Count Limiting
- [ ] 1.1 Add `-c` flag for download count (default: 1)
- [ ] 1.2 Track download count with atomic counter
- [ ] 1.3 Decrement counter on successful download
- [ ] 1.4 Shutdown server when count reaches zero
- [ ] 1.5 Handle unlimited mode (count=0)
- [ ] 1.6 Display remaining download count after each download

## 2. Directory Serving
- [ ] 2.1 Detect if path is a directory
- [ ] 2.2 Add compression flags (`-z`, `-j`, `-Z`, `-u`)
- [ ] 2.3 Implement tar.gz archive creation (default)
- [ ] 2.4 Implement ZIP archive creation (`-Z`)
- [ ] 2.5 Implement tar.bz2 archive creation (`-j`)
- [ ] 2.6 Implement uncompressed tar creation (`-u`)
- [ ] 2.7 Stream archive directly to response (avoid temp files)
- [ ] 2.8 Set appropriate filename in Content-Disposition

## 3. Upload Mode
- [ ] 3.1 Add `-U` flag for upload mode
- [ ] 3.2 Create HTML upload form
- [ ] 3.3 Implement multipart form file handler
- [ ] 3.4 Save uploaded files to current directory
- [ ] 3.5 Handle filename conflicts (append number or reject)
- [ ] 3.6 Display upload progress and completion
- [ ] 3.7 Apply download count limit to uploads

## 4. Self Distribution
- [ ] 4.1 Add `-s` flag for self-distribution mode
- [ ] 4.2 Detect own executable path
- [ ] 4.3 Serve executable with appropriate filename

## 5. Testing
- [ ] 5.1 Unit tests for download counting
- [ ] 5.2 Integration tests for directory archiving (all formats)
- [ ] 5.3 Integration tests for upload mode
- [ ] 5.4 Integration tests for self-distribution
