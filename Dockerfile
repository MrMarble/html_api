FROM golang:1.23-alpine as builder

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o main .

FROM scratch

COPY --from=builder /app/main /app/main

ENV API_PORT=8080
ENV API_HOST=0.0.0.0

CMD ["/app/main"]
