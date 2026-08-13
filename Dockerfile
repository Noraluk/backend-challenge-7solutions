FROM cosmtrek/air:v1.67.3 AS development

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

CMD ["-c", ".air.toml"]

FROM golang:1.24-alpine3.22 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY gen ./gen
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
    && addgroup -S -g 10001 app \
    && adduser -S -D -H -u 10001 -G app app

COPY --from=build --chown=app:app /out/api /usr/local/bin/api

USER app
EXPOSE 8080 9090

ENTRYPOINT ["/usr/local/bin/api"]
