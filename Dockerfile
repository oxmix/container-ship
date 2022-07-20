FROM golang:1.18-buster as builder
WORKDIR /app
COPY go.* .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ctr-ship .

FROM alpine:3.15
MAINTAINER Oxmix <oxmix@me.com>
COPY --from=builder /app/ctr-ship .
COPY web/layout.html /web/
EXPOSE 8443
ENTRYPOINT ["./ctr-ship"]