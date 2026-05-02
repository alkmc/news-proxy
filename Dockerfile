FROM golang:1.26 as builder

LABEL maintainer="Alex <github.com/alkmc>"

WORKDIR /app

COPY go.mod ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-w -s" -o news-app ./cmd/newsApp

FROM gcr.io/distroless/base-debian13
COPY --from=builder --chown=nonroot:nonroot /goapp /news-app

ENV PORT=8000

USER nonroot:nonroot

CMD ["/news-app"]
