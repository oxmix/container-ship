FROM --platform=linux/$BUILDARCH node:18-alpine as web
WORKDIR /build
COPY ./web .
RUN npm i
RUN npm run build:production

FROM golang:1.21-bookworm as app
WORKDIR /app
COPY go.* .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ctr-ship .

FROM alpine:3.18
MAINTAINER Oxmix <oxmix@me.com>
COPY --from=app /app/ctr-ship .
COPY --from=web /build/dist /web/dist
EXPOSE 8443
ENTRYPOINT ["./ctr-ship"]