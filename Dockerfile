FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY . /app

RUN go mod download
RUN go mod verify

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/bin/ ./cmd/...

FROM alpine

WORKDIR /app

COPY --from=builder /app/bin/ /app/bin/

ENTRYPOINT ["/app/bin/bot"]