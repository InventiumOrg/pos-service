-- name: CreatePOS :one
INSERT INTO pos (
    name, location, description, total_sale_unit
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: UpdatePOS :one
UPDATE pos
SET name = $2,
    location = $3,
    description = $4,
    total_sale_unit = $5
WHERE id = $1
RETURNING *;

-- name: GetPOS :one
SELECT * FROM pos
WHERE id = $1;

-- name: ListPOS :many
SELECT id, name, location, description, total_sale_unit
FROM pos
LIMIT $1 OFFSET $2;

-- name: DeletePOS :exec
DELETE FROM pos
WHERE id = $1;
