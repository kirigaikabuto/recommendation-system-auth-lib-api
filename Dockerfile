FROM golang:1.13-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o /recommendation-system-auth-lib-api

EXPOSE 8000

CMD [ "/recommendation-system-auth-lib-api" ]