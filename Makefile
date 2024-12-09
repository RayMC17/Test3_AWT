include .envrc

## run: run the cmd/api application

.PHONY : run/api
run/api:
	@echo 'Running application...'
	@go run ./cmd/api -port=4000 -env=development -limiter-burst=5 -limiter-rps=2 -limiter-enabled=true -cors-trusted-origins="http://localhost:9000 http://localhost:9001"	-db-dsn=${BOOKCLUB_DB_DSN} 


## db/psql: connect to the database using psql (terminal)
.PHONY: db/psql
db/psql:
	psql ${BOOKCLUB_DB_DSN}

## db/migrations/new name=$1: create a new database migration
.PHONY: db/migrations/new
db/migrations/new:
	@echo 'Creating migration files for ${name}...'
	migrate create -seq -ext=.sql -dir=./migrations ${name}


## db/migrations/up: apply all up database migrations
.PHONY: db/migrations/up
db/migrations/up:
	@echo 'Running up migrations ${num} times'
	migrate -path ./migrations -database ${BOOKCLUB_DB_DSN} up ${num}

## db/migrations/up: apply all up database migrations
.PHONY: db/migrations/down
db/migrations/down:
	@echo 'Running up migrations ${num} times'
	migrate -path ./migrations -database ${BOOKCLUB_DB_DSN} down ${num}

