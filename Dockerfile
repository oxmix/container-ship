FROM --platform=linux/$BUILDARCH node:23-alpine AS web
WORKDIR /build
COPY ./web .
RUN npm i
RUN npm run build:production

FROM golang:1.26-alpine AS app
WORKDIR /app
COPY ./ship/go.* .
RUN go mod download
COPY ./ship .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ship .

FROM alpine:3.23
LABEL maintainer="Oxi <oxmix@me.com>"
RUN addgroup -S app && adduser -S -D -H -G app app
COPY --from=app --chown=app:app /app/ship .
COPY --from=web --chown=app:app /build/dist /web/dist
USER app
EXPOSE 8443
CMD ["./ship"]