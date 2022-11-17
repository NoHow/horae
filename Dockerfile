# syntax=docker/dockerfile:1
FROM golang:1.19.3-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download
COPY *.go ./
COPY config.json ./
COPY private.key ./
COPY cert.pem ./

RUN go build -o /horae

EXPOSE 443

CMD [ "/horae" ]