# syntax=docker/dockerfile:1

FROM golang:1.23-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum* ./
RUN go mod download

COPY . .

RUN go build -o /mcp-arkhamdb

EXPOSE 8080

CMD [ "/mcp-arkhamdb" ]

