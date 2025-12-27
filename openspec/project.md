# Project Context

## Purpose
Simple utility for sharing files over a local network. Intended for LAN use only.

## Tech Stack
- Go (standard library only, no external dependencies)

## Project Conventions

### Code Style
- Standard Go formatting (gofmt)
- Descriptive variable names
- Single-file structure for simple utilities

### Architecture Patterns
- Single main package for CLI tools
- Handler pattern for HTTP serving
- Graceful shutdown with signal handling

### Testing Strategy
- Table-driven tests for utility functions
- Error case testing for CLI validation
- httptest package for HTTP handler testing

## Domain Context
- Ad-hoc file sharing tool for local networks
- One-shot transfers (server exits after successful download)
- No permanent service - start, share, done

## Important Constraints
- Platform support: macOS and Linux
- No authentication - simple tool for trusted LAN environments
- Standard library only - minimize dependencies

## External Dependencies
None - uses Go standard library only.
