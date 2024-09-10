# Strata

Strata is a flexible, secure, web server written in [Go](https://golang.org/) that supports features such as automatic SSL certificates, multi-source configuration, compression, CDN support and JWT-based authentication.

Warning: This server is in the initial stages of development and is not yet  complete, tested production software.

## Features

- **Automatic HTTPS** with Let's Encrypt
- **Manual SSL/TLS Certificates** support
- **HTTP-to-HTTPS redirection**
- **Multi-source Configuration** (YAML, JSON, TOML, environment variables, command-line)
- **Brotli and Gzip compression**
- **JWT-based Authentication** with OAuth2 Providers (Google, GitHub, etc.)
- **CloudFront CDN support** for static files
- **CloudWatch Logging** integration
- **Graceful Shutdown** handling for smooth server restarts

## Documents

- [Strata Server Requirements](./requirements.md)
- [Strata Product Marketing](./marketing.md)
- [Strata Installation Guide](./installation.md)

