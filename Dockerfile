FROM golang:alpine as build
RUN apk add build-base

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY main.go .
COPY utils ./utils
COPY teamspeak ./teamspeak
COPY discord ./discord

RUN go build -o /discord-teamspeak-notifier

FROM alpine

WORKDIR /

COPY --from=build /discord-teamspeak-notifier /discord-teamspeak-notifier

ENTRYPOINT ["/discord-teamspeak-notifier"]