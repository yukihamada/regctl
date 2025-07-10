# Changelog

All notable changes to regctl Cloud will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial project structure
- Cloudflare Workers-based API Gateway
- CLI tool (regctl) with device flow authentication
- Multi-registrar support (VALUE-DOMAIN, Route 53, Porkbun)
- Domain management (list, register, transfer, update)
- DNS record management (CRUD operations)
- D1 database schema for data persistence
- KV-based session caching
- Durable Objects for rate limiting
- Comprehensive API documentation
- Development setup scripts

### Security
- JWT-based authentication
- PBKDF2 password hashing
- Role-based access control
- API rate limiting

## [0.1.0] - TBD

### Added
- MVP release
- Basic domain listing from VALUE-DOMAIN
- CLI authentication
- Initial documentation

---

## Release Types

- **Major** (X.0.0) - Breaking API changes
- **Minor** (0.X.0) - New features, backwards compatible
- **Patch** (0.0.X) - Bug fixes, backwards compatible