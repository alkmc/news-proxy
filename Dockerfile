FROM golang:1.27 AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-w -s" -o newsproxy ./cmd/newsproxy

FROM gcr.io/distroless/base-debian13
COPY --from=builder /app/newsproxy /newsproxy

USER 65532:65532

CMD ["/newsproxy"]
