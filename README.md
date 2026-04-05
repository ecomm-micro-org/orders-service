# 📦 Orders Service
 
A microservice responsible for managing the full lifecycle of customer orders within the e-commerce platform. It handles order creation, retrieval, updates, and cancellations — coordinating with MongoDB for persistence, Redis for caching, and Kafka for event-driven communication with downstream services.
 
---
 
## 🏗️ Architecture Overview
 
```
┌─────────────────────────────────────────────────────────────────┐
│                        API Gateway                              │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Orders Service                             │
│                                                                 │
│   ┌─────────────┐    ┌──────────────┐    ┌──────────────────┐  │
│   │   Fiber     │───▶│  Controller  │───▶│    Use Cases     │  │
│   │   Router    │    │   Layer      │    │    / Services    │  │
│   └─────────────┘    └──────────────┘    └────────┬─────────┘  │
│                                                   │            │
│                          ┌────────────────────────┤            │
│                          │                        │            │
│                          ▼                        ▼            │
│               ┌─────────────────┐     ┌──────────────────┐    │
│               │   MongoDB       │     │    Redis Cache   │    │
│               │   Repository    │     │    Layer         │    │
│               └─────────────────┘     └──────────────────┘    │
│                                                                 │
│               ┌─────────────────────────────────────────────┐  │
│               │           Kafka Producer                    │  │
│               │  (order.created / order.cancelled /        │  │
│               │   order.updated)                           │  │
│               └─────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```
 
---
 
## 🚀 API Endpoints
 
Base path: `/api/v1/orders`
 
| Method   | Endpoint           | Description                          | Auth Required |
|----------|--------------------|--------------------------------------|---------------|
| `GET`    | `/get_my_orders`   | Get all orders for the logged-in customer | ✅ Yes   |
| `GET`    | `/order/:id`       | Get a specific order by its ID       | ✅ Yes        |
| `POST`   | `/create`          | Create a new order                   | ✅ Yes        |
| `PUT`    | `/order/:id`       | Update the delivery address of an order | ✅ Yes     |
| `DELETE` | `/order/:id`       | Cancel an order                      | ✅ Yes        |
 
---
 
### `GET /get_my_orders`
 
Returns all orders associated with the authenticated customer.
 
**Response `200 OK`**
```json
[
  {
    "id": "64f1a2b3c4d5e6f7a8b9c0d1",
    "customer_id": "usr_abc123",
    "status": "processing",
    "items": [
      { "product_id": "prod_xyz", "quantity": 2, "price": 29.99 }
    ],
    "delivery_address": {
      "street": "123 Main St",
      "city": "New York",
      "zip": "10001",
      "country": "US"
    },
    "total": 59.98,
    "created_at": "2024-09-01T10:00:00Z"
  }
]
```
 
---
 
### `GET /order/:id`
 
Fetches a single order by its unique ID. Results are cached in Redis with a configurable TTL.
 
**Path Params**
| Param | Type   | Description    |
|-------|--------|----------------|
| `id`  | string | The order's ID |
 
**Response `200 OK`**
```json
{
  "id": "64f1a2b3c4d5e6f7a8b9c0d1",
  "customer_id": "usr_abc123",
  "status": "shipped",
  "items": [...],
  "delivery_address": { ... },
  "total": 59.98,
  "created_at": "2024-09-01T10:00:00Z",
  "updated_at": "2024-09-02T08:30:00Z"
}
```
 
---
 
### `POST /create`
 
Creates a new order for the authenticated customer. On success, publishes an `order.created` event to Kafka for downstream services (e.g., Inventory, Notifications, Payment).
 
**Request Body**
```json
{
  "items": [
    { "product_id": "prod_xyz", "quantity": 2 }
  ],
  "delivery_address": {
    "street": "123 Main St",
    "city": "New York",
    "zip": "10001",
    "country": "US"
  }
}
```
 
**Response `201 Created`**
```json
{
  "id": "64f1a2b3c4d5e6f7a8b9c0d1",
  "status": "pending",
  "total": 59.98,
  "created_at": "2024-09-01T10:00:00Z"
}
```
 
---
 
### `PUT /order/:id`
 
Updates the delivery address of an order. Only allowed while the order is in `pending` or `processing` status. Publishes an `order.updated` event to Kafka.
 
**Path Params**
| Param | Type   | Description    |
|-------|--------|----------------|
| `id`  | string | The order's ID |
 
**Request Body**
```json
{
  "delivery_address": {
    "street": "456 Elm Ave",
    "city": "Brooklyn",
    "zip": "11201",
    "country": "US"
  }
}
```
 
**Response `200 OK`**
```json
{
  "id": "64f1a2b3c4d5e6f7a8b9c0d1",
  "delivery_address": { ... },
  "updated_at": "2024-09-02T09:00:00Z"
}
```
 
---
 
### `DELETE /order/:id`
 
Cancels an order. Only orders in `pending` or `processing` state can be cancelled. Publishes an `order.cancelled` event to Kafka and invalidates the Redis cache for the order.
 
**Path Params**
| Param | Type   | Description    |
|-------|--------|----------------|
| `id`  | string | The order's ID |
 
**Response `200 OK`**
```json
{
  "message": "Order cancelled successfully",
  "id": "64f1a2b3c4d5e6f7a8b9c0d1"
}
```
 
---
 
## 📊 Order Status Lifecycle
 
```
  [pending] ──────────────────────────────────▶ [cancelled]
      │                                               ▲
      ▼                                               │
 [processing] ──────────────────────────────────▶ [cancelled]
      │
      ▼
  [shipped]
      │
      ▼
 [delivered]
```
 
| Status       | Description                                    |
|--------------|------------------------------------------------|
| `pending`    | Order placed, awaiting payment confirmation    |
| `processing` | Payment confirmed, being prepared for dispatch |
| `shipped`    | Dispatched to delivery carrier                 |
| `delivered`  | Successfully delivered to customer             |
| `cancelled`  | Cancelled by customer or system                |
 
---
 
## 🛠️ Tech Stack
 
| Component     | Technology             | Purpose                                          |
|---------------|------------------------|--------------------------------------------------|
| HTTP Server   | Go Fiber               | High-performance REST API routing                |
| Database      | MongoDB                | Persistent order storage                         |
| Cache         | Redis                  | Order caching, session data, rate limiting       |
| Message Bus   | Apache Kafka           | Event streaming to downstream services           |
| Language      | Go                     | Service implementation                           |
 
---
 
## ⚙️ Configuration
 
Configuration is driven by environment variables.
 
```env
# Server
APP_PORT=8083
APP_ENV=production
 
# MongoDB
MONGO_URI=mongodb://localhost:27017
MONGO_DB_NAME=ecomm_orders
 
# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_ORDER_TTL=3600          # Cache TTL in seconds
 
# Kafka
KAFKA_BROKERS=localhost:9092
KAFKA_GROUP_ID=orders-service
KAFKA_TOPIC_ORDER_CREATED=order.created
KAFKA_TOPIC_ORDER_UPDATED=order.updated
KAFKA_TOPIC_ORDER_CANCELLED=order.cancelled
 
# Auth
JWT_SECRET=your-jwt-secret
```
 
---
 
## 📨 Kafka Events
 
The Orders Service is a **producer** for the following events:
 
### `order.created`
Published when a new order is successfully persisted.
```json
{
  "event": "order.created",
  "timestamp": "2024-09-01T10:00:00Z",
  "payload": {
    "order_id": "64f1a2b3c4d5e6f7a8b9c0d1",
    "customer_id": "usr_abc123",
    "items": [...],
    "total": 59.98
  }
}
```
 
**Consumed by:** `payment-service`, `inventory-service`, `notification-service`
 
---
 
### `order.updated`
Published when a delivery address is modified.
```json
{
  "event": "order.updated",
  "timestamp": "2024-09-02T09:00:00Z",
  "payload": {
    "order_id": "64f1a2b3c4d5e6f7a8b9c0d1",
    "updated_fields": ["delivery_address"]
  }
}
```
 
**Consumed by:** `delivery-service`, `notification-service`
 
---
 
### `order.cancelled`
Published when an order is cancelled.
```json
{
  "event": "order.cancelled",
  "timestamp": "2024-09-02T11:00:00Z",
  "payload": {
    "order_id": "64f1a2b3c4d5e6f7a8b9c0d1",
    "customer_id": "usr_abc123",
    "reason": "customer_request"
  }
}
```
 
**Consumed by:** `payment-service` (refund), `inventory-service` (restock), `notification-service`
 
---
 
## 🗄️ MongoDB Schema
 
**Collection:** `orders`
 
```json
{
  "_id": "ObjectId",
  "customer_id": "string",
  "status": "string",
  "items": [
    {
      "product_id": "string",
      "quantity": "number",
      "price": "number"
    }
  ],
  "delivery_address": {
    "street": "string",
    "city": "string",
    "zip": "string",
    "country": "string"
  },
  "total": "number",
  "created_at": "ISODate",
  "updated_at": "ISODate"
}
```
 
**Indexes:**
- `customer_id` — for fast lookup by customer
- `status` — for filtering orders by state
- `created_at` (descending) — for sorted order history
 
---
 
## 🔴 Redis Caching Strategy
 
| Key Pattern                  | Value           | TTL         | Invalidated On      |
|------------------------------|-----------------|-------------|---------------------|
| `order:{id}`                 | Order JSON      | 60 minutes  | Update / Cancel     |
| `orders:customer:{customer_id}` | Order list JSON | 5 minutes | New order / Cancel |
 
Cache is invalidated on any write operation to ensure consistency.
 
---
 
## 📁 Project Structure
 
```
orders-service/
├── cmd/
│   └── app/
│         └── app.go
│   └── main.go                  # Entry point
├── internal/
│   └── config.go                # Env config loader
├── controllers/
│   └── orders.go                # HTTP handler layer
├── routes/
│   └── order_routes.go          # Route registration (OrderRoutes)
├── services/
│   └── order_service.go         # Business logic
├── repositories/
│   └── order_repository.go      # MongoDB data access
├── cache/
│   └── redis_cache.go           # Redis caching layer
├── events/
│   └── kafka_producer.go        # Kafka event publishing
├── models/
│   └── order.go                 # Domain model definitions
├── middleware/
│   └── auth.go                   # JWT authentication middleware
├── Dockerfile
├── .env.example
└── README.md
```
 
---
 
## 🐳 Running Locally
 
### Prerequisites
- Go 1.21+
- Docker & Docker Compose
 
### Start Dependencies
 
```bash
docker-compose up -d mongo redis kafka
```
 
### Run the Service
 
```bash
cp .env.example .env
go mod tidy
go run cmd/main.go
```
 
### Run with Docker
 
```bash
docker build -t orders-service .
docker-compose up
```
 
---
 
## 🧪 Running Tests
 
```bash
# Unit tests
go test ./... -v
 
# With coverage
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```
 
---
 
## ❗ Error Responses
 
All errors follow a consistent format:
 
```json
{
  "error": {
    "code": "ORDER_NOT_FOUND",
    "message": "No order found with the given ID",
    "status": 404
  }
}
```
 
| Code                     | HTTP Status | Description                                  |
|--------------------------|-------------|----------------------------------------------|
| `ORDER_NOT_FOUND`        | 404         | Order does not exist                         |
| `UNAUTHORIZED`           | 401         | Invalid or missing JWT token                 |
| `FORBIDDEN`              | 403         | Customer does not own this order             |
| `ORDER_NOT_CANCELLABLE`  | 409         | Order status does not allow cancellation     |
| `ORDER_NOT_UPDATABLE`    | 409         | Order status does not allow address updates  |
| `VALIDATION_ERROR`       | 400         | Missing or invalid request body fields       |
| `INTERNAL_ERROR`         | 500         | Unexpected server error                      |
 
---
 
## 🔗 Related Services
 
| Service               | Interaction                                               |
|-----------------------|-----------------------------------------------------------|
| `auth-service`        | JWT validation for all authenticated routes               |
| `payment-service`     | Consumes `order.created` to process payment               |
| `inventory-service`   | Consumes `order.created` / `order.cancelled` for stock    |
| `notification-service`| Consumes all order events to send customer notifications  |
| `delivery-service`    | Consumes `order.updated` to update shipment details       |