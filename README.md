# Orders Service

A gRPC microservice that manages the full order lifecycle for an e-commerce platform. It handles order creation, payment processing via Razorpay, delivery address management, and real-time order streaming. It integrates with a separate Products service, publishes events to Kafka, and sends Slack notifications on order activity.

---

## Table of Contents

- [Architecture](#architecture)
- [Prerequisites](#prerequisites)
- [Environment Variables](#environment-variables)
- [Running Locally](#running-locally)
- [Docker](#docker)
- [API Reference](#api-reference)
- [Project Structure](#project-structure)
- [Testing](#testing)
- [CI/CD](#cicd)

---

## Architecture

The service is built as a gRPC server with unary and streaming endpoints. All inbound calls are validated by a JWT auth interceptor before reaching the business logic layer. A logging interceptor wraps every call and records method, duration, and gRPC status code via Zap.

```
Client
  |
  | gRPC
  v
Auth Interceptor (JWT validation)
  |
Logging Interceptor (Zap)
  |
Order Handlers
  |
  |--- Order Service
          |--- MongoDB (persistence)
          |--- Kafka Producer (orders.created event)
          |--- Products gRPC Client (price calculation)
          |--- Razorpay Client (payment order creation)
          |--- Slack Messenger (order notifications)
```

---

## Prerequisites

- Go 1.25 or later
- MongoDB
- Redis
- Kafka
- A Razorpay account (test or live)
- A Slack bot token and channel ID

---

## Environment Variables

| Variable | Description |
|---|---|
| `DSN` | MongoDB connection string |
| `CACHE_ADDR` | Redis address (e.g. `localhost:6379`) |
| `CACHE_PASSWD` | Redis password |
| `BROKERS` | Comma-separated list of Kafka broker addresses |
| `PORT` | Server port (e.g. `:42067`) |
| `SECRET_KEY` | HMAC secret used to validate JWT tokens |
| `RAZORPAY_KEY_ID` | Razorpay API key ID |
| `RAZORPAY_SECRET` | Razorpay API secret |
| `COURIER_KEY` | Courier notification API key |
| `PRODUCTS_CLIENT` | Address of the Products gRPC service (e.g. `localhost:42069`) |
| `SLACK_TOKEN` | Slack bot OAuth token |
| `SLACK_CHANNEL` | Slack channel ID to receive order notifications |

---

## Running Locally

Install dependencies:

```bash
go mod download
```

Set environment variables and start the server:

```bash
export DSN=mongodb://root:mongo_pass@localhost:27017/ordersDB?authSource=admin
export CACHE_ADDR=localhost:6379
export CACHE_PASSWD=yourpassword
export BROKERS=localhost:9092
export PORT=:42067
export SECRET_KEY=yoursecretkey
export RAZORPAY_KEY_ID=rzp_test_xxxx
export RAZORPAY_SECRET=yoursecret
export COURIER_KEY=yourcourierkey
export PRODUCTS_CLIENT=localhost:42069
export SLACK_TOKEN=xoxb-xxxx
export SLACK_CHANNEL=CXXXXXXXXXX

go run main.go
```

Or use the provided shell script:

```bash
chmod +x run.sh
./run.sh
```

The server listens on TCP port `42067`.

---

## Docker

Build the image:

```bash
docker build -t orders-service .
```

Run the container:

```bash
docker run -p 42067:42067 \
  -e DSN=mongodb://root:mongo_pass@mongo:27017/ordersDB?authSource=admin \
  -e CACHE_ADDR=redis:6379 \
  -e CACHE_PASSWD=yourpassword \
  -e BROKERS=kafka:9092 \
  -e PORT=:42067 \
  -e SECRET_KEY=yoursecretkey \
  -e RAZORPAY_KEY_ID=rzp_test_xxxx \
  -e RAZORPAY_SECRET=yoursecret \
  -e COURIER_KEY=yourcourierkey \
  -e PRODUCTS_CLIENT=products-service:42069 \
  -e SLACK_TOKEN=xoxb-xxxx \
  -e SLACK_CHANNEL=CXXXXXXXXXX \
  orders-service
```

The multi-stage Dockerfile compiles the binary on a Go builder image and produces a minimal Alpine image for deployment.

---

## API Reference

All methods are defined in `gen/pb/orders.proto` and served over gRPC on the registered service `orders.OrdersService`.

Every endpoint except `GetKey` requires a valid JWT passed as gRPC metadata:

```
Authorization: Bearer <token>
```

| Method | Type | Description |
|---|---|---|
| `GetKey` | Unary | Returns the Razorpay public key ID for client-side checkout initialisation |
| `CreateOrder` | Unary | Creates a new order, registers a Razorpay payment order, and publishes an `orders.created` Kafka event |
| `GetOrderByID` | Unary | Fetches a single order by its ID; only accessible by the order owner |
| `GetOrdersByCustomerID` | Server streaming | Streams all orders belonging to the authenticated customer |
| `UpdateDeliveryAddress` | Unary | Updates the delivery address for an existing order; only accessible by the order owner |
| `PaymentSuccess` | Unary | Not yet implemented |
| `PaymentFailure` | Unary | Not yet implemented |
| `PaymentCallback` | Unary | Not yet implemented |
| `CancelOrder` | Unary | Not yet implemented |

---

## Project Structure

```
orders/
  api/                  gRPC server setup and service registration
  cache/                Redis client initialisation and helpers
  db/                   MongoDB client initialisation and helpers
  gen/pb/               Generated protobuf and gRPC code
  handlers/             gRPC handler implementations
  interceptors/         Auth and logging gRPC interceptors
  internal/
    auth/               JWT validation and claims types
    config/             Environment variable loading
    kafka/              Kafka producer and topic definitions
    messaging/          Slack messenger
  models/               Order domain model
  services/             Order business logic
  store/                Storer interface, MongoStore, and MemoryStore
  main.go               Application entry point
  Dockerfile            Multi-stage Docker build
  run.sh                Local development helper script
```

---

## Testing

Run all tests:

```bash
go test -vet=off ./...
```

All tests use the `testify` package (`require` and `assert`). External dependencies such as MongoDB and Kafka are not required at test time; the service layer and handlers are tested against `store.MemoryStore`, a thread-safe in-memory implementation of the `store.Storer` interface located at `store/memory_store.go`.

Known issue: the standard `go test ./...` command fails a vet check due to a `fmt.Sprintf` call with arguments but no formatting directive in `handlers/handlers.go:70`. No production code was changed to work around this. Pass `-vet=off` to skip the vet step when running tests locally.

---

## CI/CD

The GitHub Actions workflow at `.github/workflows/main.yaml` runs on every push and pull request to `main`.

It has two jobs:

1. `test` - sets up Go 1.25 and downloads dependencies.
2. `build-and-push` - builds a multi-platform Docker image using Buildx and pushes it to Docker Hub when the trigger is a push to `main` (not a pull request).

The Docker Hub image is tagged as `latest` on the default branch and with a `sha-` prefixed commit SHA on every run.

Required repository secrets:

| Secret | Description |
|---|---|
| `DOCKERHUB_USERNAME` | Your Docker Hub username |
| `DOCKERHUB_TOKEN` | A Docker Hub access token with write permission |
