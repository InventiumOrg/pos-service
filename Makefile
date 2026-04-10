postgres:
	podman run --network inventium --name postgres -e POSTGRES_USER="$(PG_USER)" -e POSTGRES_PASSWORD="$(PG_PASSWORD)" -p 5432:5432 -d postgres:16-alpine
createdb:
	podman exec -it postgres createdb --username="$(PG_USER)" --owner="$(PG_USER)" pos-service
dropdb:
	podman exec -it postgres dropdb --username="$(PG_USER)" pos-service
migrateup:
	migrate -path ./models/migration -database "$(DB_SOURCE)" -verbose up
migratedown:
	migrate -path ./models/migration -database "$(DB_SOURCE)" -verbose down
sqlc:
	sqlc generate --no-remote
loaddata:
	PGPASSWORD="$(DB_PASSWORD)" psql -h "$(DB_HOST)" -p 16677 -U "$(DB_USER)" -d pos_service -f data/sql/inventium.sql
runcontainer:
	podman run --network inventium --name pos-service -p 11570:11570 -d -e DB_SOURCE="$(DB_SOURCE)" -e CLERK_KEY="$(CLERK_KEY)" -e SERVICE_NAME="$(SERVICE_NAME)" pos-service:1.0.0
.PHONY: postgres createdb dropdb migrateup migratedown sqlc loaddata runcontainer