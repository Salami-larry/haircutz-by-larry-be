# Admin user (manual seed)

Insert one document into MongoDB collection **`admins`** (database name from `MONGODB_URI`, e.g. `haircutz`).

There is **no signup**. Seed admins manually (share credentials as needed).

## Fields

| Field | Description |
|-------|-------------|
| `email` | Login email (unique, stored lowercase at login match) |
| `passwordHash` | bcrypt hash of password — never store plaintext |
| `role` | Must be `"admin"` for dashboard access |
| `createdAt` | UTC date |

## Generate a password hash

```bash
cd haircutz-by-larry-be
go run ./scripts/hashpwd.go 'your-plain-password'
```

Or with Python:

```bash
python3 -c "import bcrypt; print(bcrypt.hashpw(b'your-plain-password', bcrypt.gensalt(rounds=12)).decode())"
```

## Example insert (mongosh)

```javascript
use haircutz

db.admins.insertOne({
  email: "admin@haircutz.local",
  passwordHash: "$2a$12$REPLACE_WITH_GENERATED_HASH",
  role: "admin",
  createdAt: new Date()
})
```

Only users with **`role: "admin"`** receive a JWT from `POST /api/v1/admin/login` and can call protected admin routes.

## Login

```http
POST /api/v1/admin/login
Content-Type: application/json

{"email":"admin@haircutz.local","password":"your-plain-password"}
```

Response includes `token`. Use on protected routes:

```http
Authorization: Bearer <token>
GET /api/v1/admin/me
```

The middleware checks the JWT and that **`role` in the token is `admin`** (copied from the DB at login).
