# Application Architecture

## Overview

The user-management service follows a small hexagonal architecture. HTTP and gRPC are inbound adapters, while MongoDB, bcrypt, and JWT are outbound adapters. Application services coordinate business operations through interfaces in `internal/ports`; `internal/domain` contains the user entity and domain errors.

The dependency direction is toward the core:

```text
                         cmd/api (composition root)
                                    |
                                    v
     +----------------------------------------------------------+
     | Adapters                                                 |
     | HTTP handlers/routes | gRPC server | MongoDB | bcrypt/JWT|
     +--------------------------+-------------------------------+
                                |
                                v
     +----------------------------------------------------------+
     | Application services + ports                             |
     | registration | authentication | user CRUD | interfaces   |
     +--------------------------+-------------------------------+
                                |
                                v
     +----------------------------------------------------------+
     | Domain                                                   |
     | User entity | domain errors                              |
     +----------------------------------------------------------+
```

At runtime, inbound and outbound flows are different even though their source-code dependencies point inward:

```text
HTTP/gRPC -> application use case -> repository/hash/token port -> MongoDB/bcrypt/JWT adapter
```

## Package responsibilities

| Package | Responsibility |
| --- | --- |
| `cmd/api` | Composition root, configuration, dependency wiring, server lifecycle, and shutdown |
| `internal/domain` | User entity and errors meaningful to the business |
| `internal/application` | Registration, login, CRUD orchestration, validation calls, and the user-count worker |
| `internal/application/dto` | Use-case inputs and sanitized user results |
| `internal/ports` | Interfaces for use cases, persistence, password hashing, and tokens |
| `internal/adapters/http` | HTTP routes, handlers, authentication middleware, JSON transport, and HTTP error mapping |
| `internal/adapters/grpc` | Protobuf RPC handlers and Bearer metadata interceptor |
| `internal/adapters/mongodb` | MongoDB connection, indexes, persistence models, and repository implementation |
| `internal/adapters/auth` | bcrypt password hashing and HS256 JWT generation/validation |
| `internal/platform` | Environment configuration parsing and validation |
| `gen/user/v1` | Generated protobuf messages and gRPC service bindings |

## Dependency inversion and repository abstraction

Application services depend on `ports.UserRepository`, not on the MongoDB driver. The MongoDB adapter implements that interface, and `cmd/api` supplies the implementation when constructing services. This keeps persistence details out of use cases and enables unit tests to use generated mocks.

The tradeoff is additional interfaces and mapping code. For this service, the boundary is kept small: one user repository plus focused authentication and use-case ports.

## Unique email enforcement

MongoDB creates the unique ascending index `users_email_unique` on `users.email` during startup. Database enforcement is required because an application-level check followed by insert would race under concurrent registrations. The repository translates duplicate-key failures into `domain.ErrEmailAlreadyExists`.

Startup fails if the index cannot be created. This favors data correctness over partial availability and also means existing duplicate data must be cleaned before the service can start successfully.

## Password security

Registration hashes passwords with bcrypt's default cost before persistence. Login compares the supplied password with the stored hash. The domain entity contains `PasswordHash`, but application responses and protobuf `User` messages omit it.

Bcrypt deliberately consumes CPU to slow password guessing. Its cost may need tuning for production latency and hardware; this implementation uses the library default.

## JWT security and authorization scope

Login issues an HS256 JWT containing the user ID as `sub`, plus `iat` and `exp`. Validation restricts the accepted algorithm to HS256 and requires issued-at, expiration, and a non-empty subject. HTTP middleware reads the `Authorization: Bearer <token>` header; the gRPC interceptor reads equivalent `authorization` metadata. Both reuse the same `TokenService`.

Registration and login are public. HTTP `/users` routes and gRPC `GetUser` are protected. Authentication is not resource authorization: the subject is placed in context, but handlers do not compare it with the requested user ID. Consequently, any authenticated user can currently read, update, or delete any user. Roles, ownership rules, refresh tokens, revocation, key rotation, and TLS are outside the implemented scope.

## HTTP and gRPC reuse

The HTTP registration handler and gRPC `CreateUser` both call `RegistrationUseCase`. HTTP user handlers and gRPC `GetUser` call `UserUseCase`. Password hashing, validation, and persistence therefore remain in application services instead of being duplicated in transport adapters.

`application/dto.UserResponse` maps sanitized results to protobuf because the project intentionally exposes `user.ToProto()`. This is convenient and keeps the gRPC adapter small, but it couples an application DTO to generated transport code. A stricter hexagonal design would keep that mapping inside the gRPC adapter.

The HTTP boundary maps validation and domain errors to stable status codes and sanitized JSON messages. The gRPC handlers currently return application errors directly, so gRPC uses its default conversion rather than a dedicated domain-to-status mapper. This is a known tradeoff of the current implementation.

## Background worker lifecycle

`RunUserCountWorker` uses a 10-second ticker and the repository count operation. `cmd/api` derives its context from the process signal context. During shutdown it cancels that context and waits for the worker to exit, preventing the goroutine and ticker from being left behind.

HTTP shuts down with a timeout, gRPC uses graceful stop, and MongoDB disconnects with its own timeout. The current gRPC graceful stop has no forced-stop timeout, so a long-running RPC could delay shutdown.

## Error boundaries

Domain errors identify expected outcomes such as not found, duplicate email, and invalid credentials. Application services wrap infrastructure failures with `%w`, retaining cause information for boundary inspection.

The HTTP adapter is the public error boundary: it converts known errors into `400`, `401`, `404`, or `409` responses and hides unknown details behind `500 INTERNAL_ERROR`. Internal errors are logged server-side. MongoDB-specific errors do not escape as HTTP response text.

As noted above, gRPC does not currently have an equivalent status-code mapper. Clients should not depend on raw gRPC error messages until that boundary is introduced.

## Composition and testability

`cmd/api` constructs one repository, one bcrypt service, one token service, and the application services, then shares those instances between HTTP and gRPC. This guarantees both transports operate on the same data and authentication rules.

Ports are mocked with `mockgen` for application and adapter unit tests. MongoDB collection behavior is abstracted for repository tests. The repository can be verified with `make check`; protobuf contracts are checked separately with `make proto`.
