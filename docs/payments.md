# Payments (Paystack) — Phase 4

## Flow

1. `POST /api/v1/appointments` → `booked` + `paystackReference` (`hbl_…`)
2. `POST /api/v1/payments/initialize` `{ "appointmentId": "…" }` → access code
3. Client Paystack Inline (Phase 5 storefront) or admin **Mark paid**
4. Webhook `POST /api/v1/webhooks/paystack` (`charge.success`) and/or `GET /api/v1/payments/verify?reference=`
5. Status → `paid`, assign `HBL-YYYYMMDD-XXXXXX`, email customer + admin once

## Endpoints

| Method | Path | Auth |
|--------|------|------|
| POST | `/api/v1/payments/initialize` | Public |
| GET | `/api/v1/payments/verify?reference=` | Public |
| POST | `/api/v1/payments/abandon` `{reference, email}` | Public |
| POST | `/api/v1/webhooks/paystack` | HMAC signature |
| POST | `/api/v1/admin/appointments/:id/mark-paid` | Admin JWT |

Abandoned unpaid holds can resume via initialize. Payment wins over abandon.

Callback URL defaults to `{CLIENT_PUBLIC_URL}/book/success`. Origin-only `PAYSTACK_CALLBACK_URL` values also append `/book/success`.

## Manual mark paid

Admin UI: **Appointments → Open → Mark paid** (booked or abandoned). Assigns tracking + sends paid emails once.

## Smoke checks

1. Create a hold: `POST /api/v1/appointments` (Phase 3).
2. Initialize: `POST /api/v1/payments/initialize` with `{ "appointmentId" }` → access code / reference.
3. Pay in Paystack test mode, or skip to step 5.
4. Verify: `GET /api/v1/payments/verify?reference=…` → `paid` + `HBL-…`.
5. Or admin Mark paid on the appointment detail page.
6. Confirm customer + `ADMIN_NOTIFY_EMAIL` received confirmation (SMTP).
7. Client: open `{CLIENT}/book/success?reference=…` and confirm tracking shown.

