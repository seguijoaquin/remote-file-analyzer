FROM golang:alpine

WORKDIR /app
COPY daemon/config.go /app/
COPY daemon/config.json /app/
COPY daemon/dtos.go /app/
COPY daemon/daemon.go /app/

RUN go build

CMD ./app