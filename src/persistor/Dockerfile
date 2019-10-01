FROM golang:alpine

WORKDIR /app
COPY persistor/persistor.go /app/

RUN go build

CMD ./app