# Build stage
FROM golang:1.18.10-alpine3.17 AS builder
WORKDIR /app
COPY . .
RUN export GOPROXY=https://goproxy.cn && go build -ldflags="-s -w" -o gohotdeploy .

# Run stage
FROM alpine:3.14
WORKDIR /app
COPY --from=builder /app/gohotdeploy .
COPY config.yml ./config/config.yml

VOLUME config
EXPOSE 8080

CMD [ "/app/gohotdeploy", "--config", "/app/config/config.yml" ]