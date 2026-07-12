# news-proxy

A server-side rendered search interface for [NewsAPI](https://newsapi.org/), built with Go's standard library only.

## Features

- Article search with pagination, capped at the NewsAPI free tier limit (100 results)
- Dark/light theme with system preference detection
- Security headers (CSP, nosniff) and structured request logging
- Templates and static assets embedded in a single binary
- Graceful shutdown
- Zero external dependencies

## Environment Variables

| Variable       | Description                                              | Default | Required |
|----------------|----------------------------------------------------------|---------|----------|
| `NEWS_API_KEY` | API key from [NewsAPI.org](https://newsapi.org/register) | -       | **Yes**  |
| `PORT`         | The port the server listens on                           | `8080`  | No       |

## How to run

Once started, the app is available at <http://localhost:8080>.

### Option 1: Docker (Recommended)

```bash
export NEWS_API_KEY="your_api_key_here"
make up
```

Alternatively, put the key in an `.env` file next to `compose.yaml`, Docker Compose picks it up automatically.

*To stop the container, run `make down`.*

### Option 2: Build from source

Requires Go 1.26+:

```bash
export NEWS_API_KEY="your_api_key_here"
make run
```

## Development

```bash
make test       # run tests with the race detector
make lint       # golangci-lint
make fmt        # gofumpt
make deadcode   # find unreachable functions
```
