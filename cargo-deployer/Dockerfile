FROM alpine:3.15
RUN apk --no-cache add --update php7 php-json php-curl php-pcntl php-openssl docker-cli pciutils && \
	wget -q -O /etc/apk/keys/sgerrand.rsa.pub https://alpine-pkgs.sgerrand.com/sgerrand.rsa.pub && \
	wget https://github.com/sgerrand/alpine-pkg-glibc/releases/download/2.33-r0/glibc-2.33-r0.apk && \
    apk --no-cache add glibc-2.33-r0.apk && rm glibc-2.33-r0.apk
WORKDIR /app
COPY . .
CMD ["/app/tasks.php"]