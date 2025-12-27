# Change: Add File Uploading

## Why

Users need to receive files from others, not just share files. An upload mode allows others to send files to the user via a simple web form.

## What Changes

- Add upload mode for receiving files from others via HTTP upload forms
- Apply download count limit to uploads (limit number of uploads accepted)
- Save uploaded files to current directory

## Impact

- Affected specs: `file-uploading` (new capability)
- Affected code: Add upload handler, HTML form, multipart form processing
- Dependencies: Go standard library (net/http, mime/multipart)
