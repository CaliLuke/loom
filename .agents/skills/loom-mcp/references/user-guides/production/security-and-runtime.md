# Production: Security And Runtime

Use this for production auth defaults, graceful shutdown, config, and server timeouts.

## Security

- Prefer API-level security defaults
- Use `NoSecurity()` explicitly for public methods
- Always use HTTPS in production
- Rotate secrets and keys
- Log auth failures
- Still validate input on authenticated endpoints

## Graceful Shutdown

Use signal handling plus `server.Shutdown(...)` with a timeout.

## Configuration

Prefer environment-based config for:

- HTTP/gRPC listen addresses
- database URLs
- log level
- read/write timeouts

## HTTP Server Timeouts

Set at least:

- `ReadHeaderTimeout`
- `ReadTimeout`
- `WriteTimeout`
- `IdleTimeout`
- `MaxHeaderBytes`
