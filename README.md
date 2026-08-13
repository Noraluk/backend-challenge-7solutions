# Backend Golang Coding Test

## Overview

This coding test has two parts.

| Section | Focus | Submission Type |
| --- | --- | --- |
| User Management API | Build a Golang user management API with MongoDB and JWT authentication | Code implementation |
| Lottery Search System | Design a real-world lottery ticket search solution with wildcard matching | Design proposal only (no code) |

## User Management API

### Objective (User Management API)

Build a RESTful API in Golang to manage users, using MongoDB for persistence, JWT for authentication, and clean code practices.

### Requirements (User Management API)

#### 1. User Model

Define a user entity with the following fields:

- `ID` (auto-generated)
- `Name` (string)
- `Email` (string, unique)
- `Password` (hashed)
- `CreatedAt` (timestamp)

#### 2. Authentication

Implement:

- User registration
- User authentication that returns a JWT token

JWT requirements:

- Protect endpoints with JWT
- Validate tokens via middleware
- Sign tokens using HMAC (`HS256`) with a secret key

#### 3. User Operations

Implement the following operations:

- Create a new user
- Fetch a user by ID
- List all users
- Update a user's name or email
- Delete a user

#### 4. MongoDB Integration

- Use the official Go MongoDB driver
- Persist and retrieve user data from MongoDB

#### 5. Middleware

- Implement logging middleware to capture HTTP method, path, and execution time

#### 6. Concurrency Task

- Run a background goroutine every 10 seconds to log the total number of users in the database

#### 7. Testing

- Write unit tests using Go's standard `testing` package
- Mock MongoDB interactions where appropriate

### Bonus (Optional, User Management API)

- **Containerization**: Add Docker and `docker-compose` support for the API and MongoDB
- **Abstraction**: Use Go interfaces to abstract MongoDB operations for better testability
- **Validation**: Implement input validation (for example, required fields and email format)
- **Graceful Shutdown**: Handle system signals using `context.Context`
- **gRPC Support**:
  - Define a `.proto` file for `CreateUser` and `GetUser`
  - Implement a gRPC server (optionally secure with token metadata)
- **Hexagonal Architecture**:
  - Structure the project using ports and adapters
  - Separate domain, application, and infrastructure layers
  - Decouple business logic from frameworks and drivers

### Deliverables (User Management API)

Provide a Git repository containing:

- `README.md` with setup and execution instructions
- A guide explaining how to generate and use JWT tokens
- Sample API requests and responses
- Documentation of assumptions or design decisions

### Evaluation Criteria (User Management API)

- Code quality, structure, and readability
- Correctness and completeness of the REST API
- Security and implementation of JWT
- Proper usage and abstraction of MongoDB
- Test coverage and effective mocking
- Idiomatic Go usage
- Bonus implementations (gRPC, Docker, validation, architecture)

## Lottery Search System

### Objective (Lottery Search System)

Design a real-world solution to search a large dataset of lottery tickets using pattern matching with wildcard support.

> This section is a design exercise. Do not implement code.

### Requirements (Lottery Search System)

#### 1. Data Volume

- Handle a dataset of **10 million** lottery tickets
- Each ticket is a 6-digit number

#### 2. Search Pattern

- Support a 6-character search pattern containing digits and wildcards (`*`)
- Example patterns:

| Pattern | Matches |
| --- | --- |
| `****23` | Numbers ending in `23` |
| `1****5` | Numbers starting with `1` and ending with `5` |
| `123***` | Numbers starting with `123` |

#### 3. Result Distribution

- Constraint: the same search pattern should not return the same ticket to multiple users at the same time
- Propose a distribution mechanism so matching tickets are assigned without duplicate simultaneous selection

#### 4. Performance

- Ensure the search is performant for `10M+` records
- Propose an efficient approach for querying and allocation

#### 5. Real-World Design Proposal (No Code Required)

- Recommend the database/storage technology you would use in production and explain why
- Describe the algorithm and indexing strategy used for wildcard pattern matching
- Explain how you would prevent duplicate simultaneous results for the same pattern (for example, locking, reservation, or atomic allocation)
- No code implementation is required; provide a solution/design only

### Deliverables (Lottery Search System)

Submit a design document only (no code implementation) that includes:

- Proposed solution architecture, data structures, and algorithms
- Recommended production database/storage choice with justification (for example, query performance, concurrency handling, operational simplicity)
- Performance analysis summarizing efficiency and tradeoffs
- Concurrency/distribution strategy explaining how duplicate results are avoided for the same pattern

### Evaluation Criteria (Lottery Search System)

- Feasibility: the solution addresses the stated requirements
- Performance: the search approach is efficient for the target scale
- Correctness: the distribution constraint is handled correctly
- Real-world practicality: the database/storage and concurrency approach are appropriate for production use
- Creativity: thoughtful use of data structures and algorithms

---

# Reviewer Guide

This guide is appended after the original challenge statement so the original content remains unchanged. For design details, see [Application Architecture](docs/architecture.md).

## Prerequisites

The recommended setup only requires:

- Docker with Docker Compose
- `curl` for the HTTP examples

For local development without running the API in Docker, install Go 1.24 or newer and make MongoDB available. `grpcurl` or Postman is optional for testing gRPC.

## Configuration

Copy the example configuration before starting the application:

```sh
cp .env.example .env
```

Set `JWT_SECRET` in `.env` to a non-production value containing at least 32 characters, for example:

```dotenv
JWT_SECRET=local-review-only-secret-at-least-32-bytes
```

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `HTTP_PORT` | No | `8080` | HTTP server port |
| `GRPC_PORT` | No | `9090` | gRPC server port |
| `MONGO_URI` | Yes | None | MongoDB connection URI |
| `MONGO_DATABASE` | Yes | None | MongoDB database name |
| `JWT_SECRET` | Yes | None | HS256 signing key; minimum 32 characters |
| `JWT_TTL` | Yes | None | Token lifetime as a Go duration, such as `1h` |
| `MONGO_PORT` | No | `27017` | Host port published by Docker Compose |

`.env` is ignored by Git. `.env.example` contains placeholders only and must not contain a real secret.

## Start with Docker

After configuring `.env`, build and start the API and MongoDB:

```sh
docker compose up -d --build
docker compose ps
```

The API is ready when both services are healthy. Verify the HTTP server:

```sh
curl http://localhost:8080/health
```

Expected response:

```text
ok
```

View logs or stop the stack with:

```sh
docker compose logs -f api
docker compose down
```

`docker compose down` preserves MongoDB data. Use `make compose-reset` only when the local MongoDB volume should also be removed.

## Start locally

Start only MongoDB in Docker:

```sh
docker compose up -d mongodb
```

Export configuration for a process running on the host, then start the API:

```sh
export HTTP_PORT=8080
export GRPC_PORT=9090
export MONGO_URI=mongodb://localhost:27017
export MONGO_DATABASE=user_management
export JWT_SECRET=local-review-only-secret-at-least-32-bytes
export JWT_TTL=1h
make run
```

The process serves HTTP on port `8080` and gRPC on port `9090`. Press `Ctrl+C` to trigger graceful shutdown.

## HTTP API

`POST /auth/register`, `POST /auth/login`, and `GET /health` are public. Every `/users` route requires this header:

```text
Authorization: Bearer <access_token>
```

### Register a user

Registration is the HTTP create-user operation; there is intentionally no separate `POST /users` route.

```sh
curl -i http://localhost:8080/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"name":"Ada Lovelace","email":"ada@example.com","password":"local-password"}'
```

Status: `201 Created`

```json
{
  "id": "507f1f77bcf86cd799439011",
  "name": "Ada Lovelace",
  "email": "ada@example.com",
  "created_at": "2026-08-13T10:30:00Z"
}
```

The password is hashed before persistence and is never returned.

### Login and obtain a JWT

```sh
curl -i http://localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"ada@example.com","password":"local-password"}'
```

Status: `200 OK`

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

Copy `access_token` and use it in subsequent examples:

```sh
export TOKEN='paste-access-token-here'
export USER_ID='507f1f77bcf86cd799439011'
```

### Get a user

```sh
curl -i "http://localhost:8080/users/${USER_ID}" \
  -H "Authorization: Bearer ${TOKEN}"
```

Status: `200 OK`. The response uses the same user shape returned by registration.

### List users

```sh
curl -i http://localhost:8080/users \
  -H "Authorization: Bearer ${TOKEN}"
```

Status: `200 OK`

```json
[
  {
    "id": "507f1f77bcf86cd799439011",
    "name": "Ada Lovelace",
    "email": "ada@example.com",
    "created_at": "2026-08-13T10:30:00Z"
  }
]
```

### Update a user

Send `name`, `email`, or both:

```sh
curl -i -X PATCH "http://localhost:8080/users/${USER_ID}" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Ada Byron","email":"ada.byron@example.com"}'
```

Status: `200 OK`. The response contains the updated user.

### Delete a user

```sh
curl -i -X DELETE "http://localhost:8080/users/${USER_ID}" \
  -H "Authorization: Bearer ${TOKEN}"
```

Status: `204 No Content` with an empty response body.

### HTTP status and error reference

| Situation | Status | Error code |
| --- | --- | --- |
| Invalid JSON or unknown request field | `400 Bad Request` | `INVALID_REQUEST` |
| Invalid input | `400 Bad Request` | `VALIDATION_ERROR` |
| Invalid user ID | `400 Bad Request` | `INVALID_USER_ID` |
| Missing or invalid Bearer token | `401 Unauthorized` | `UNAUTHORIZED` |
| Invalid login credentials | `401 Unauthorized` | `INVALID_CREDENTIALS` |
| Duplicate email | `409 Conflict` | `EMAIL_ALREADY_EXISTS` |
| User not found | `404 Not Found` | `USER_NOT_FOUND` |
| Unexpected failure | `500 Internal Server Error` | `INTERNAL_ERROR` |

Errors use this shape and do not expose internal details:

```json
{
  "code": "USER_NOT_FOUND",
  "message": "user not found"
}
```

## gRPC bonus

The protobuf contract is at `proto/user/v1/user.proto`. The server does not enable reflection, so clients must load that file. Generate and validate Go bindings with:

```sh
make proto-generate
make proto
```

Available RPCs:

| RPC | Access |
| --- | --- |
| `user.v1.UserService/CreateUser` | Public |
| `user.v1.UserService/GetUser` | Bearer metadata required |

Create a user:

```sh
grpcurl -plaintext -import-path proto -proto user/v1/user.proto \
  -d '{"name":"Grace Hopper","email":"grace@example.com","password":"local-password"}' \
  localhost:9090 user.v1.UserService/CreateUser
```

Get a user using the JWT returned by HTTP login:

```sh
grpcurl -plaintext -import-path proto -proto user/v1/user.proto \
  -H "authorization: Bearer ${TOKEN}" \
  -d "{\"id\":\"${USER_ID}\"}" \
  localhost:9090 user.v1.UserService/GetUser
```

In Postman, create a gRPC request for `localhost:9090`, import `proto/user/v1/user.proto`, and add `authorization: Bearer <access_token>` under Metadata for `GetUser`.

## Verification

```sh
make proto
make check
docker compose config --quiet
```

`make check` runs unit tests, race detection, `go vet`, and a production build. Mocks can be regenerated with `make mocks`.

## Assumptions and known tradeoffs

- HTTP registration is the create-user operation; a duplicate `POST /users` endpoint is not provided.
- Registration and login are public. All `/users` HTTP routes and gRPC `GetUser` require authentication.
- Authentication verifies the JWT subject, but resource ownership and role-based authorization are not implemented. Any authenticated user can operate on any user ID.
- User listing is not paginated because the challenge scope does not define pagination.
- Email uniqueness is enforced by a MongoDB unique index. Concurrent duplicates are handled by the database.
- JWTs use HS256 and expire according to `JWT_TTL`. TLS, refresh tokens, key rotation, and token revocation are outside the current scope.
- gRPC shares the HTTP application services and repository. Reflection and TLS are not enabled, and application errors currently use gRPC's default error conversion.
- The background worker logs the user count every 10 seconds and stops with the application context.
- Example credentials and secrets in this guide are for local review only.
