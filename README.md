# Haircutz by Larry — API

Go (Gin) backend for **Haircutz by Larry** — MongoDB, Paystack, SMTP, and Supabase Storage.

Phase 0 ships health/readiness only. Domain features land in later phases.

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
| `JWT_SECRET` | Phase 1+ | Secret for admin JWT |
| `SUPABASE_*` | Phase 2+ | Haircutz Supabase project (not Home Essentials) |
| `PAYSTACK_SECRET_KEY` | Phase 4+ | Paystack secret |
| `CLIENT_PUBLIC_URL` | no | Storefront URL (default `http://localhost:3000`) |
| `PAYSTACK_CALLBACK_URL` | no | Defaults to `{CLIENT_PUBLIC_URL}/book/success` |
| `SMTP_*` / `ADMIN_NOTIFY_EMAIL` | Phase 4+ | Outbound mail |
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

## Tests

```bash
go test ./...
```
