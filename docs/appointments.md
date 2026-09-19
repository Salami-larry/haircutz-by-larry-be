# Appointments & availability

Timezone: **Africa/Lagos**. Slot step: **15 minutes**. Unpaid `booked` holds expire after **15 minutes** → `abandoned` (no email).

## Hours

| | Walk-in | Home service |
|--|---------|--------------|
| Mon–Sat | 09:00–22:00 | 10:00–17:00 |
| Sunday | 12:00–22:00 | closed |

## Public

### Availability

```http
GET /api/v1/availability?date=2026-09-20&hairstyleId=<id>&serviceType=walk_in
```

`serviceType`: `walk_in` | `home_service`

Response:

```json
{
  "date": "2026-09-20",
  "serviceType": "walk_in",
  "durationMinutes": 45,
  "slots": ["2026-09-20T09:00:00+01:00", "..."],
  "closed": false
}
```

### Create hold

```http
POST /api/v1/appointments
```

```json
{
  "hairstyleId": "...",
  "serviceType": "walk_in",
  "startAt": "2026-09-20T09:00:00+01:00",
  "customer": {
    "name": "Ada",
    "email": "ada@example.com",
    "phone": "+234...",
    "address": "",
    "notes": ""
  }
}
```

Home service requires `address`. Status starts as `booked`. Overlap → **409** `slot_unavailable`.

### Track

```http
POST /api/v1/appointments/track
{ "trackingNumber": "HBL-…", "email": "…" }
```

Wrong pair → **404**. Response includes `canReschedule` when status is `missed`.

### Reschedule (missed only, free)

```http
POST /api/v1/appointments/:id/reschedule
{ "trackingNumber": "HBL-…", "email": "…", "startAt": "RFC3339" }
```

Same start/end → **409** `same_timeframe`. Success → status `paid`, emails customer + admin. No Paystack.

## Admin

| Method | Path | Notes |
|--------|------|--------|
| GET | `/api/v1/admin/appointments?status=&date=&page=&page_size=` | Inbox |
| GET | `/api/v1/admin/appointments/:id` | Detail |
| POST | `/api/v1/admin/appointments/:id/mark-paid` | `booked`\|`abandoned` → `paid` (+ tracking + paid emails) |
| PATCH | `/api/v1/admin/appointments/:id/status` | Allow-list transitions |

### Status transitions (Phase 6)

| From | To | How |
|------|-----|-----|
| `booked` / `abandoned` | `paid` | Mark paid (or Paystack) |
| `paid` | `acknowledged` | PATCH `{ "status": "acknowledged" }` |
| `acknowledged` | `completed` \| `missed` | PATCH |

Illegal jumps → **409** `invalid_status_transition`.

Emails: **completed** → customer only; **missed** → customer only (track/reschedule CTA). No email on **acknowledged**.

Blocking statuses for the calendar: `booked`, `paid`, `acknowledged`.

## Background jobs (Phase 8)

In-process tickers (cancelled on API shutdown):

| Job | Rule |
|-----|------|
| Abandon holds | `booked` older than 15m → `abandoned` |
| T−15m reminder | `paid`\|`acknowledged`, `startAt` within next 15m, `reminderSentAt` empty; **skipped** if created with &lt;15m lead; emails customer + admin once |
| Post-end nag | `paid`\|`acknowledged` with `endAt` in the past; admin email every **30m** via `postTimeNagAt` until `completed`\|`missed` |
