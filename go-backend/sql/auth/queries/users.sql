-- name: CreateUser :one
INSERT INTO auth.users (email, password_hash, role) 
VALUES ($1, $2, $3) 
RETURNING *;

-- name: GetUserByEmail :one
SELECT a.id,
       a.email,
       a.password_hash,
       a.created_at,
       a.role 
  FROM auth.users a
WHERE a.email = $1 
LIMIT 1;

-- name: UpdateUserToRegistered :one
UPDATE auth.users
   SET email = $1,
       password_hash = $2,
       role = 'user',
       updated_at = CURRENT_TIMESTAMP
 WHERE id = $3
RETURNING *;
