# uServe

Share a file over your local network with a single command.

## Installation

```bash
go install github.com/mtymek/userve@latest
```

Or build from source:

```bash
git clone https://github.com/mtymek/userve.git
cd userve
go build -o userve .
```

## Usage

```bash
userve <file>
```

This starts a temporary HTTP server and displays a URL. Share the URL with someone on your network - once they download the file, the server automatically exits.

### Options

```
-p <port>    Port to listen on (default: 8080)
-i <ip>      IP address to bind to (default: all interfaces)
```

### Examples

```bash
# Share a file on the default port
userve document.pdf

# Use a custom port
userve -p 9000 photo.jpg

# Bind to a specific interface
userve -i 192.168.1.100 archive.zip
```

## Inspiration

This project is inspired by Woof (https://github.com/simon-budig/woof) tool, which doesn't seem to be maintained anymore.
