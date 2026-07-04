FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o expense-tracker .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/expense-tracker .
EXPOSE 8080
CMD ["./expense-tracker"]