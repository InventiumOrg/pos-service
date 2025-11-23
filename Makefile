postgres:
	podman run --network inventium --name postgres-1 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -p 5432:5432 -d postgres:16-alpine
createdb:
	podman exec -it postgres-1 createdb --username=root --owner=root pos-service
dropdb:
	podman exec -it postgres-1 dropdb --username=root pos-service
migrateup:
	migrate -path ./models/migration -database "$(DB_SOURCE)" -verbose up
migratedown:
	migrate -path ./models/migration -database "$(DB_SOURCE)" -verbose down
sqlc:
	sqlc generate --no-remote
loaddata:
	PGPASSWORD=secret psql -h localhost -U root -d inventium -f data/sql/inventium.sql
runcontainer:
	podman run --network inventium --name pos-service -p 7450:7450 -d -e DB_SOURCE="$(DB_SOURCE)" -e CLERK_KEY="$(CLERK_KEY)" pos-service:1.0.0
.PHONY: postgres createdb dropdb migrateup migratedown sqlc loaddata runcontainer