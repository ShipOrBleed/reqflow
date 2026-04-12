FROM golang:1.22-alpine

RUN apk add --no-cache git bash jq
RUN go install github.com/zopdev/govis/cmd/govis@latest

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
