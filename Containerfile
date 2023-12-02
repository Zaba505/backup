FROM alpine:latest

# Install 1password CLI
RUN echo https://downloads.1password.com/linux/alpinelinux/stable/ >> /etc/apk/repositories
RUN wget https://downloads.1password.com/linux/keys/alpinelinux/support@1password.com-61ddfc31.rsa.pub -P /etc/apk/keys
RUN apk update && apk add 1password-cli

# Install restic CLI
RUN apk add restic

COPY ./backup /usr/local/bin

ENTRYPOINT ["backup"]