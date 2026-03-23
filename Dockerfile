FROM mirror.gcr.io/library/golang:1.22-alpine AS builder

WORKDIR /app
COPY . .

RUN go build -o tina-ddns .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/tina-ddns .
COPY config.json .

CMD ["./tina-ddns"]
