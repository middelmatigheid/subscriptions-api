# SUBSCRIPTIONS

[![Go](https://img.shields.io/badge/Go-123?logo=go)](https://go.dev/)
[![Gin](https://img.shields.io/badge/Gin-123?logo=gin)](https://gin-gonic.com/en/)
[![Database](https://img.shields.io/badge/Postgres-119?logo=postgresql)](https://www.postgresql.org/)
[![Cache](https://img.shields.io/badge/Redis-900?logo=redis)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-purple?logo=docker)](https://www.docker.com/)
[![Goose](https://img.shields.io/badge/Goose-123?logo=go)](https://github.com/pressly/goose)
[![Swagger](https://img.shields.io/badge/Swagger-191?logo=swagger)](https://swagger.io/)
[![REST_API](https://img.shields.io/badge/REST_API-white)](https://en.wikipedia.org/wiki/REST)

<img src="screenshots/screenshot-1.png">

# About the project

It is a simple restful api for subscriptions management that was done within test assignment. There are three-layer architecture **handlers → service → database**, server's graceful shutdown, database migrations, pagination, filtering and indexing, caching. The swagger documentation was also provided

# Main functions

- Create subscription
- Read subscription
- Update subscription
- Partial update subscription
- Delete subscription
- Get list of subscriptions
- Get total price of subscriptions

# Used in project

- Go
- Gin
- PostgreSQL
- Redis
- Goose
- Swagger
- Slog
- Docker

# Downloading and running the api

### 1. Run Docker

Install and run Docker on your computer


### 2. Build up the images

```bash
docker compose down -v
docker system prune -a -f
docker compose build --no-cache
docker compose up
```

### 2. Migrate the database using goose

Open new terminal and migrate the database

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
goose postgres "user=postgres password=password dbname=subscriptions host=localhost port=5433" up -dir ./migrations
goose postgres "user=postgres password=password dbname=subscriptions host=localhost port=5433" status -dir ./migrations
```

### 3. The api is ready

The api would be available via http://localhost:8080/subscriptions/swagger/index.html

# Project structure

```bash
subscriptions/
├── cmd/server/main.go          # Server to run
├── internal/                 
│   ├── handlers/handlers.go    # Handlers package for handling requests with gin
│   ├── service/service.go      # Service package for business logic
│   ├── database/database.go    # Database package for operating with PostgreSQL
│   ├── cache/cache.go          # Cache package for redis caching
│   ├── models/models.go        # Models package
│   └── config/config.go        # Config package
├── migrations/                 # SQL migrations
├── docs/                       # Swagger docs
├── dockerfile
├── dockerignore
├── docker-compose.yml         
├── go.mod
├── go.sum
└── .env
```
