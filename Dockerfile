FROM golang:1.26 AS builder

LABEL maintainer="Alex <github.com/alkmc>"

WORKDIR /app

COPY go.mod ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-w -s" -o news-proxy ./cmd/newsproxy

FROM gcr.io/distroless/base-debian13
COPY --from=builder --chown=nonroot:nonroot /app/news-proxy /news-proxy

USER nonroot:nonroot

CMD ["/news-proxy"]
