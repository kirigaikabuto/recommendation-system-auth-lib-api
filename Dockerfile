FROM golang:1.13-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./
COPY *.env ./

RUN go build -o /rec-api

EXPOSE 8000

CMD [ "/rec-api -c=prod.env" ]