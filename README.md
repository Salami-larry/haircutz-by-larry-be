# Haircutz by Larry — API

Go (Gin) backend for **Haircutz by Larry** — MongoDB, Paystack, SMTP, and Supabase Storage.

## Prerequisites

- Go 1.22+
- MongoDB (Atlas or local)

## Environment

Create `.env` in this folder and fill in values (never commit `.env`).

| Variable | Required | Description |
|----------|----------|-------------|
| `HTTP_ADDR` | no | Listen address (default `:8080`) |
| `MONGODB_URI` | **yes** | Mongo URI including DB name (e.g. `.../haircutz`) |
| `CORS_ORIGINS` | no | Comma-separated origins (default `http://localhost:3000,http://localhost:3001`) |
| `JWT_SECRET` | yes | Secret for admin JWT |
| `SUPABASE_*` | for uploads | Haircutz Supabase project |
| `PAYSTACK_SECRET_KEY` | for payments | Paystack secret |
| `CLIENT_PUBLIC_URL` | no | Storefront URL (default `http://localhost:3000`) |
| `PAYSTACK_CALLBACK_URL` | no | Defaults to `{CLIENT_PUBLIC_URL}/book/success` |
| `SMTP_*` / `ADMIN_NOTIFY_EMAIL` | for email | Outbound mail + admin notify address |
| `APP_LOG_FILE` | no | Optional log file path |

## Run locally

```bash
# ensure .env exists with at least MONGODB_URI
go mod tidy
go run .
```

- Health: [http://localhost:8080/health](http://localhost:8080/health)
- Ready: [http://localhost:8080/ready](http://localhost:8080/ready)

Seed an admin — [docs/admin-seed.md](docs/admin-seed.md).

Hairstyles CRUD — [docs/hairstyles.md](docs/hairstyles.md).

Appointments & availability — [docs/appointments.md](docs/appointments.md).

Payments — [docs/payments.md](docs/payments.md).

E2E QA — [docs/e2e-checklist.md](docs/e2e-checklist.md).

## Tests

```bash
go test ./...
```
