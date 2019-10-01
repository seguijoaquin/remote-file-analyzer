FROM golang:alpine

WORKDIR /app
COPY processor/processor.go /app/

RUN go build

CMD ./app