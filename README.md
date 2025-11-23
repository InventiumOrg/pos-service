# P.O.S Service
Repository for P.O.S Service

Product Journey:
https://varsentinel.atlassian.net/wiki/spaces/Inventiums/overview

## Using Gin Framework:

https://gin-gonic.com/en/docs/quickstart/

## SQLC:

https://docs.sqlc.dev/en/stable/index.html

## Project Structures:
```
    pos-service/
    ├── api/                   # Gin Router Controller
    │     ├── server.go        #
    ├── config/                # Store Applications Configs
    │     ├── config.go        #
    ├── handlers/              # Handlers Controller for different API Methods
    │     └── pos.go           #
    ├── middlewares/           # Middlewares to check foe authorized client
    |     └── authenticate.go  #
    |── models/                # Models for working with Postgresl
    |     |── migration        # DB Migration
    |     |── query.           # DB Query
    |     |── sqlc             # DB Connection
    |── routes/                # Stores Route
    |     └── routes.go        #
```
## API Routes

- List POS:   /pos
- Get POS:    /pos/:id
- Create POS: /pos/:id
- Update POS: /pos/:id
- Delete POS: /pos/:id

## Usage

How to perform db migration:

Prerequisites:
- Set $DB_SOURCE to the PostgreSQL URL

Run the following DB Migration Steps:
- For DB Migration Up
```
    $ make migrateup
```
- For DB Migration Down
```
    $ make migratedown
```
Run this command to generate sqlc code
```
    $ make sqlc
```