# Golang Weather Application

Weather application with OpenWeather API integration, user authentication, and location management.

## Features

- User registration and authentication with sessions
- Location management (create, list, delete)
- OpenWeather API integration
  - Location search by city name
  - Weather data retrieval by coordinates
  - **Rate limiting (60 requests/minute) with automatic request queueing**
- PostgreSQL database with connection pooling
- HTTP server with middleware support
- Structured logging with Zap

## OpenWeather API Rate Limiting

The application implements intelligent rate limiting for OpenWeather API requests to comply with the free tier limit of 60 requests per minute.

### How It Works

- **Token Bucket Algorithm**: Continuously refills at 60 tokens/minute
- **Automatic Queueing**: Excess requests are queued (up to 100)
- **Context-Aware**: Respects request timeouts and cancellations
- **Thread-Safe**: Handles concurrent requests safely

### Testing

### Unit Tests

The project includes comprehensive unit test coverage for critical components:

#### OpenWeather HTTP Transport Tests

- **Location**: `internal/features/openweather/transport/http/`
- **Command**: `make test-openweather`
- **Coverage**:
  - ✅ Successful API requests (location search, weather by coordinates)
  - ✅ HTTP error handling (4xx, 5xx status codes)
  - ✅ Invalid JSON response handling
  - ✅ Rate limiter integration
  - ✅ Network error handling

The tests use mocked HTTP servers and services to ensure fast, reliable, and isolated testing without external dependencies.

For detailed information, see: [OpenWeather Transport Tests README](internal/features/openweather/transport/http/README_TESTS.md)

### Integration Tests

**Note**: Tests require an OpenWeather API key. Set `OPENWEATHER_API_KEY` in `.env.test` or they will skip automatically.

### Running Tests

```bash
# Run all tests
make test

# Run integration tests only
make test-integration

# Run specific feature tests
make test-users
make test-sessions
make test-openweather

# Run all tests with coverage
make test-all
```

## Configuration

Set these environment variables in your `.env` file:

```env
OPENWEATHER_API_BASE_URL=https://api.openweathermap.org
OPENWEATHER_API_KEY=your_api_key_here
OPENWEATHER_REQUESTS_PER_MINUTE=60  # Optional, default: 60
OPENWEATHER_QUEUE_SIZE=100           # Optional, default: 100
```

### Error Responses

- **429 Too Many Requests**: Rate limit exceeded, request queued
- **503 Service Unavailable**: Queue is full
- **408 Request Timeout**: Request timed out while waiting in queue

All error responses include retry information:

```json
{
  "error": "Rate limit exceeded",
  "message": "Too many requests. Rate limit: 60 requests per minute.",
  "rate_limit": {
    "requests_per_minute": 60,
    "retry_after_seconds": 5
  }
}
```

## Environment Variables

See `.env.example` for all available configuration options:

- **HTTP Server**: `HTTP_ADDR`, `HTTP_SHUTDOWN_TIMEOUT`, `ALLOWED_ORIGINS`
- **PostgreSQL**: `POSTGRES_HOST`, `POSTGRES_PORT`, `POSTGRES_USER`, etc.
- **Logger**: `LOGGER_LEVEL`, `LOGGER_FOLDER`
- **Sessions**: `SESSION_TTL`
- **OpenWeather**: `OPENWEATHER_API_BASE_URL`, `OPENWEATHER_API_KEY`, `OPENWEATHER_REQUESTS_PER_MINUTE`, `OPENWEATHER_QUEUE_SIZE`

## Building and Running

```bash
# Copy example environment file
cp .env.example .env

# Edit .env with your configuration
# Add your OpenWeather API key

# Build
go build -o weather ./cmd/weather

# Run
./weather
```

## API Endpoints

### Authentication

- `POST /api/v1/users/sign-up` - Register new user
- `POST /api/v1/users/sign-in` - User login
- `POST /api/v1/users/sign-out` - User logout

### Locations

- `GET /api/v1/locations` - List user locations (requires auth)
- `POST /api/v1/locations` - Create location (requires auth)
- `DELETE /api/v1/locations/:id` - Delete location (requires auth)

### Weather

- `POST /api/v1/open-weather/get-location` - Search location by city name (requires auth)
- `POST /api/v1/open-weather/get-weather` - Get weather by coordinates

## Project Structure

```
.
├── cmd/weather/              # Application entry point
├── internal/
│   ├── core/                 # Core infrastructure
│   │   ├── domain/          # Domain models
│   │   ├── errors/          # Error types
│   │   ├── logger/          # Logging
│   │   ├── repository/      # Database connection
│   │   └── transport/       # HTTP server & middleware
│   └── features/            # Feature modules
│       ├── locations/       # Location management
│       ├── openweather/     # OpenWeather API integration
│       │   ├── ratelimiter/ # Rate limiting (NEW)
│       │   ├── service/     # Business logic
│       │   └── transport/   # HTTP handlers
│       ├── sessions/        # Session management
│       └── users/           # User management
└── .kiro/specs/             # Feature specifications
    └── openweather-rate-limiting/  # Rate limiting spec
```

## OpenWeather API Integration Tests

Comprehensive integration test suite for OpenWeather API integration:

- ✅ Configuration loading and validation
- ✅ Location search API integration
- ✅ Weather data API integration
- ✅ Rate limiter behavior (60 requests/minute)
- ✅ Timeout handling
- ✅ Context cancellation
- ✅ Concurrent request handling
- ✅ Response data validation

See [Integration Tests Documentation](README_INTEGRATION_TESTS.md) for detailed information.

### Quick Start

```bash
# Run all OpenWeather integration tests
make test-openweather

# Run short tests (skip long-running tests)
make test-openweather-short

# Run with coverage
make test-openweather-coverage
```

**Note**: Tests require an OpenWeather API key. Set `OPENWEATHER_API_KEY` in `.env.test` or they will skip automatically.

## Rate Limiting Implementation Details

For detailed information about the rate limiting implementation, see:

- [Requirements](.kiro/specs/openweather-rate-limiting/requirements.md)
- [Design](.kiro/specs/openweather-rate-limiting/design.md)
- [Implementation Guide](.kiro/specs/openweather-rate-limiting/README.md)

## License

MIT
