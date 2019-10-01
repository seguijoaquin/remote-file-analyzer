FROM golang:alpine

WORKDIR /app
COPY status/config.go /app/
COPY status/config.json /app/
COPY status/dtos.go /app/
COPY status/status.go /app/

RUN go build

CMD ./app