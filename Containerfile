FROM golang:1.21-alpine as build
WORKDIR /crud

COPY . .

RUN go mod download
RUN go build -o /backup -ldflags="-s -w" main.go

FROM alpine:latest

WORKDIR /

# Install 1password CLI
RUN echo https://downloads.1password.com/linux/alpinelinux/stable/ >> /etc/apk/repositories
RUN wget https://downloads.1password.com/linux/keys/alpinelinux/support@1password.com-61ddfc31.rsa.pub -P /etc/apk/keys
RUN apk update && apk add 1password-cli

# Install restic CLI
RUN apk add restic

COPY --from=build /backup /backup

ENTRYPOINT ["/backup"]