FROM golang:alpine

WORKDIR /app
COPY persistor/config.go /app/
COPY persistor/config.json /app/
COPY persistor/dtos.go /app/
COPY persistor/persistor.go /app/

RUN go build

CMD ./app