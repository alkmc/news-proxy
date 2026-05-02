# news-proxy

A web proxy and search interface for the [NewsAPI](https://newsapi.org/), serving HTML templates rendered with Go's standard library.

## Environment Variables

| Variable       | Description                   | Default | Required |
|----------------|-------------------------------|---------|----------|
| `NEWS_API_KEY` | Your API key from NewsAPI.org | -       | **Yes**  |
| `PORT`         | The port the server listens on| `3000`  | No       |

## How to run

### Option 1: Docker (Recommended)

Use the provided Makefile to easily build and run the application in the background:

```bash
make docker-up
```

*To stop the container, run `make docker-down`.*

### Option 2: Build from source

Requires Go 1.26+. Set your API key and use the Makefile:

```bash
export NEWS_API_KEY="your_api_key_here"

# Build and run the app
make run
```
