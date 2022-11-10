/* PRICE */
-- name: InsertPrice :one
INSERT INTO price (asset_hash,amount,basis_point) VALUES ($1,$2,$3) RETURNING *;

-- name: UpdatePrice :exec
UPDATE price SET amount=$1,basis_point=$2 WHERE asset_hash=$3;

-- name: GetPrice :one
SELECT * FROM price WHERE asset_hash=$1;

/* TOPUP */
-- name: InsertTopup :one
INSERT INTO topup (id,status,partial_tx,asset_hash,asset_amount,asset_price,fee_amount,requested_at,completed_at,settled_at,expire_at,spread_amount)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING *;

-- name: UpdateTopup :exec
UPDATE topup SET status=$1,partial_tx=$2,asset_amount=$3,asset_price=$4,fee_amount=$5,completed_at=$6,settled_at=$7,expire_at=$8,spread_amount=$9 WHERE id=$10;

-- name: GetTopup :one
SELECT * FROM topup WHERE id=$1;

-- name: GetAllTopups :many
SELECT * FROM topup;

/* WHITELIST */
-- name: InsertWhitelist :one
INSERT INTO whitelist (script) VALUES ($1) RETURNING *;

-- name: GetWhitelists :many
SELECT * FROM whitelist;

-- name: DeleteWhitelists :exec
DELETE FROM whitelist WHERE script IN ($1);


