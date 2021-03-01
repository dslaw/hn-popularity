FROM golang:1.15-alpine AS builder

# Install `go-sqlite3` dependencies.
RUN apk add --update gcc musl-dev

WORKDIR /go/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN go build -o /app/producer .


FROM alpine:3.12.1 AS release
COPY --from=builder /app/producer /app/producer

VOLUME /app/data

CMD ["/app/producer"]
