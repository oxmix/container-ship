FROM golang:1.22-alpine AS app
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o deployer .

FROM alpine:3.18
LABEL maintainer="Oxi <oxmix@me.com>"
RUN apk --no-cache add docker-cli
COPY --from=app /app/deployer .
CMD ["./deployer"]

#FROM alpine:3.15
#RUN apk --no-cache add --update php7 php-json php-curl php-pcntl php-openssl docker-cli pciutils && \
#	wget -q -O /etc/apk/keys/sgerrand.rsa.pub https://alpine-pkgs.sgerrand.com/sgerrand.rsa.pub && \
#	wget https://github.com/sgerrand/alpine-pkg-glibc/releases/download/2.33-r0/glibc-2.33-r0.apk && \
#    apk --no-cache add glibc-2.33-r0.apk && rm glibc-2.33-r0.apk
#WORKDIR /app
#COPY . .
#CMD ["/app/tasks.php"]
