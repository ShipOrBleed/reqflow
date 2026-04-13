FROM golang:1.25-alpine

RUN apk add --no-cache git bash jq
RUN go install github.com/thzgajendra/govis/cmd/govis@latest

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
