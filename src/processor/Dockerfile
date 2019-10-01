FROM golang:alpine

WORKDIR /app
COPY processor/config.go /app/
COPY processor/config.json /app/
COPY processor/dtos.go /app/
COPY processor/processor.go /app/

RUN go build

CMD ./app