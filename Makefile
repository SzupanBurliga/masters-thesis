.PHONY: monitoring monitoring-down up down test fmt build seed proto gql certs tokens psql

COMPOSE = docker compose -f infra/docker-compose.yml

up:
	$(COMPOSE) up -d postgres

down:
	$(COMPOSE) --profile rest --profile graphql --profile grpc down -v

build:
	$(COMPOSE) --profile rest --profile graphql --profile grpc build

seed:
	cd app && go run ./cmd/seed

test:
	cd app && go test ./...

proto:
	cd app && protoc --proto_path=api --go_out=. --go_opt=module=thesis/app --go-grpc_out=. --go-grpc_opt=module=thesis/app api/service.proto

gql:
	cd app && go run github.com/99designs/gqlgen generate

certs:
	infra/certs/gen.sh

tokens:
	cd app && go run ./cmd/tokens -n 100 -out ../loadtests/tokens.json

fmt:
	cd app && gofmt -l -w .

psql:
	psql postgres://postgres:test@localhost:5432/shop


# Podgląd na żywo: Grafana http://localhost:3000, Prometheus http://localhost:9090 (opcjonalny, obciąża maszynę)
monitoring:
	$(COMPOSE) --profile monitoring up -d

monitoring-down:
	$(COMPOSE) --profile monitoring stop prometheus grafana
	$(COMPOSE) --profile monitoring rm -f prometheus grafana
