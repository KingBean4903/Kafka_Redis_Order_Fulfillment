# Dockerfile

FROM golang:1.24.4-bookworm

WORKDIR /app

COPY go.sum go.mod ./

RUN go mod download

COPY . .

RUN go build -o order-processor ./services/order-processor

CMD ["./order-processor"]
