# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
ENV CGO_ENABLED=0 GOOS=linux
RUN go build -ldflags "-s -w" -o server .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && addgroup -S app && adduser -S app -G app
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=builder /app/server /app/server
COPY --from=builder /app/config /app/config
EXPOSE 8081
USER app
ENTRYPOINT ["/app/server"]
