FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod tidy
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -o orders 

FROM alpine:3.14
WORKDIR /app
COPY --from=builder /app/orders .
EXPOSE 42067
CMD ["./orders"]
