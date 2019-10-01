FROM golang:alpine

WORKDIR /app
COPY status/status.go /app/

RUN go build

CMD ./app