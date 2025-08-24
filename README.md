# Strava Webhook Proxy

A Go-based proxy server that facilitates Strava webhook integration by providing OAuth authentication, webhook subscription management, and event forwarding capabilities.

## Features

- **Web Interface**: Simple HTML frontend for managing Strava connections and webhook subscriptions
- **OAuth Integration**: Complete Strava OAuth 2.0 flow for athlete authorization
- **Webhook Management**: Create and manage Strava webhook subscriptions
- **Event Forwarding**: Automatically forward webhook events to multiple configured endpoints
- **Authentication Support**: Optional authentication headers for forwarded webhooks
- **Manual Webhook Testing**: Endpoint for manually sending webhook payloads

## Quick Start

1. **Set up environment variables** (see Configuration section below)
2. **Run the server**:
   ```bash
   go run .
   ```
3. **Access the web interface** at `http://localhost:8080`

## Configuration

### Required Environment Variables

| Variable | Description |
|----------|-------------|
| `STRAVA_CLIENT_ID` | Your Strava application's client ID |
| `STRAVA_CLIENT_SECRET` | Your Strava application's client secret |
| `WEBHOOK_BASE_URL` | Base URL where your server is accessible (e.g., `https://yourdomain.com`) |
| `FORWARD_URLS` | Comma-separated list of URLs to forward webhook events to |

### Optional Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `STRAVA_WEBHOOK_VERIFY_TOKEN` | Custom verification token for webhooks | `STRAVA_WEBHOOK_VERIFY_TOKEN` |
| `AUTH_HEADER_NAME` | Header name for authentication (when forwarding) | None |
| `AUTH_HEADER_TOKEN` | Authentication token to include in forwarded requests | None |
| `UI_AUTH_TOKEN` | Required token for accessing the web interface | `default_token` |

### Example .env file

```env
STRAVA_CLIENT_ID=your_client_id
STRAVA_CLIENT_SECRET=your_client_secret
WEBHOOK_BASE_URL=https://your-domain.com
FORWARD_URLS=https://api.example.com/webhook,https://backup.example.com/strava
PORT=8080
STRAVA_WEBHOOK_VERIFY_TOKEN=your_verify_token
AUTH_HEADER_NAME=Authorization
AUTH_HEADER_TOKEN=your_token_here
UI_AUTH_TOKEN=your_secure_ui_password
```

## API Endpoints

### Web Interface
- `GET /login` - Authentication page for UI access
- `POST /auth` - Authentication endpoint for login validation
- `GET /` - Main web interface for managing Strava connections (requires authentication)

### OAuth Flow
- `GET /auth/callback` - Handles OAuth callback from Strava

### Webhook Management
- `POST /create-subscription` - Creates a new Strava webhook subscription
- `GET /webhook` - Webhook verification endpoint (used by Strava)
- `POST /webhook` - Receives webhook events from Strava

### Manual Testing
- `POST /sendToWebhook` - Manually send webhook payload to configured endpoints

## Strava Application Setup

1. Create a Strava application at https://www.strava.com/settings/api
2. Set the authorization callback domain to match your `WEBHOOK_BASE_URL`
3. Note your Client ID and Client Secret for the environment configuration

## Webhook Event Flow

1. Strava sends webhook events to `/webhook`
2. Events are logged and parsed
3. Raw event JSON is forwarded to all URLs in `FORWARD_URLS`
4. If authentication is configured, appropriate headers are added

## Authentication for Forwarded Webhooks

When `AUTH_HEADER_NAME` is set, the proxy will include the specified authentication header in all forwarded webhook requests:

```env
AUTH_HEADER_NAME=X-API-Key
AUTH_HEADER_TOKEN=your-secret-key
```

This will add the header `X-API-Key: your-secret-key` to all forwarded webhook requests.

## UI Authentication

The web interface is protected by token-based authentication to prevent unauthorized access.

### Features
- **Token Authentication**: Requires `UI_AUTH_TOKEN` to access the web interface
- **Rate Limiting**: Maximum of 3 login attempts per IP address
- **Automatic Blocking**: IPs are blocked for 15 minutes after 3 failed attempts
- **Session Management**: Successful login creates a session cookie valid for 1 hour

### Access Flow
1. Navigate to your server URL (e.g., `http://localhost:8080`)
2. You'll be redirected to `/login` if not authenticated
3. Enter your `UI_AUTH_TOKEN` value
4. Upon successful authentication, you'll be redirected to the main interface

### Security Notes
- Set a strong, unique value for `UI_AUTH_TOKEN`
- The login page shows remaining attempts after failed logins
- Session cookies are HTTP-only for security
- All protected endpoints require authentication

## Development

### Dependencies
- Go 1.24.3+
- github.com/gorilla/mux v1.8.1

### Running Locally
```bash
go mod tidy
go run .
```

### Building for Production
```bash
go build -o strava-webhook-proxy
./strava-webhook-proxy
```

## Deployment

This application can be deployed to any platform that supports Go applications:

- **Heroku**: Set environment variables in the dashboard
- **Railway**: Configure environment variables in project settings
- **Docker**: Create a Dockerfile and set environment variables
- **Traditional VPS**: Build binary and run with environment variables

Ensure your `WEBHOOK_BASE_URL` points to your deployed application's URL.

## Troubleshooting

### Common Issues

1. **"BAD_URL" in logs**: `WEBHOOK_BASE_URL` environment variable is not set
2. **Webhook verification fails**: Check that `STRAVA_WEBHOOK_VERIFY_TOKEN` matches what you configured
3. **OAuth redirect issues**: Ensure your Strava app's callback domain matches your deployment URL
4. **Forward requests failing**: Verify URLs in `FORWARD_URLS` are accessible and properly formatted

### Logging

The application provides detailed logging for:
- Webhook verification requests
- Incoming webhook events
- Event forwarding attempts
- OAuth flow steps
- Configuration warnings
