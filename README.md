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

curl --location --request GET 'http://localhost:11890/api/v1/pos/list ' \
--header 'Authorization: Bearer eyJhbGciOiJSUzI1NiIsImNhdCI6ImNsX0I3ZDRQRDExMUFBQSIsImtpZCI6Imluc18yclhPVXZhbHRjWHBqSTJRQUg3WFZFTUlRNWkiLCJ0eXAiOiJKV1QifQ.eyJleHAiOjE3NjI2MzM3MTUsImZ2YSI6Wzk5OTk5LC0xXSwiaWF0IjoxNzYyNjMwMTE1LCJpc3MiOiJodHRwczovL2FjZS1sb3VzZS00Mi5jbGVyay5hY2NvdW50cy5kZXYiLCJuYmYiOjE3NjI2MzAxMDUsIm9yZ19pZCI6Im9yZ18zMGNJY090cUtIVFpvTWpOYVUxQ0h1dmRsd3kiLCJvcmdfcGVybWlzc2lvbnMiOltdLCJvcmdfcm9sZSI6Im9yZzptZW1iZXIiLCJvcmdfc2x1ZyI6InRlc3QtMTc1MzkyMDUxMiIsInNpZCI6InNlc3NfMzVEMjJXdU5IaGNVVFRYWlhOQkZ0U0NMS3FsIiwic3RzIjoiYWN0aXZlIiwic3ViIjoidXNlcl8zMGNIUVVIU3pYVDJ6Y3lGdU81Snc5emxoZHMifQ.n-28p1U263O5H53cQFXxcOPZ70QBc8yPjyj6eMQuh66Hz0yzeIpc3By03hi1fVPi1cwYi6xqlEpS9hvkJXbphHR5PnN_NW3_QVsC6UKbcA2p74t9ZKcylgErg3rDLUjP3rQsTc-K7rU9TgXbMCawO7A_t4D7MMXOwqYIH1w96NU6FUDcrleQu8IAXHDfVczj4-eJShSntWZbvZ5J2da2Q_FzkcwBf2NTNGi891_39bPdg0iDEaLoEDvdeH3BZqWuUhnOKvCz_nrWlnSunObnmNTfpH5fT5j-Z9cihuKV87O1wZ-hWtdt5BjaDzQ3yr4nye488C3qubJ6Bqx_FV0oyg'