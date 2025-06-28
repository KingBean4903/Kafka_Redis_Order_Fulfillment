# Dockerfile

FROM golang:1.24.4-bookworm

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o order-ingester ./services/order-ingester

CMD ["./order-ingester"]
