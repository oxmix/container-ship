FROM --platform=linux/$BUILDARCH node:20-alpine AS web
WORKDIR /build
COPY ./web .
RUN npm i
RUN npm run build:production

FROM golang:1.22-alpine AS app
WORKDIR /app
COPY ./ship/go.* .
RUN go mod download
COPY ./ship .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ship .

FROM alpine:3.18
LABEL maintainer="Oxi <oxmix@me.com>"
COPY --from=app /app/ship .
COPY --from=web /build/dist /web/dist
EXPOSE 8443
CMD ["./ship"]