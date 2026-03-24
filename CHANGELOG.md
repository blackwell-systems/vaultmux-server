# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Security

- **Dependency Updates**: Updated Go to 1.26.1 and quic-go to v0.57.0 to incorporate latest security patches and improvements
- **Input Validation**: Added comprehensive validation for secret names with allowlist-based character restrictions to enhance data integrity
- **Rate Limiting**: Implemented token bucket rate limiting (100 RPS) to protect against excessive request patterns
- **Security Headers**: Added standard HTTP security headers (X-Content-Type-Options, X-Frame-Options, X-XSS-Protection, Strict-Transport-Security, Content-Security-Policy) following OWASP recommendations

### Testing

- **Test Coverage**: Expanded test suite from 2.3% to 80%+ coverage with comprehensive integration tests for HTTP handlers, improving reliability and maintainability
- Added test utilities and mock implementations for isolated handler testing
- Added comprehensive tests for structured logging and error sanitization

### Changed

- Go version requirement updated from 1.24 to 1.26.1
- CI workflows updated to use Go 1.26.1
- **Structured Logging**: Replaced unstructured log.Printf with go.uber.org/zap for production-ready JSON logging with structured fields
- **Error Handling**: Backend errors are now sanitized before being sent to clients, preventing information disclosure while maintaining full internal logging
- **Production Mode**: Gin framework now runs in release mode by default (configurable via GIN_MODE), disabling debug output and improving performance
- Dockerfile updated to explicitly set GIN_MODE=release for production deployments

### Added

- New `pkg/logger` package for centralized structured logging with production and development modes
- Error sanitization function that logs full error details internally while returning generic messages to clients

---

## Note on Security Posture

These changes represent proactive security hardening and observability improvements identified through internal code audit as part of our ongoing commitment to production readiness. The updates align with industry best practices, OWASP security guidelines, and twelve-factor app methodology for production operations. While the previous implementation was functional for development and testing environments, these enhancements bring the codebase to production-grade standards with proper observability, error handling, and operational safety.
