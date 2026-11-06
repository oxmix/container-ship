FROM golang:1.26-alpine AS app
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o deployer .

FROM alpine:3.23
LABEL maintainer="Oxi <oxmix@me.com>"
RUN apk --no-cache add docker-cli
COPY --from=app /app/deployer .
CMD ["./deployer"]
