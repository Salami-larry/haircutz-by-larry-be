# Appointments & availability (Phase 3)

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

## Admin

| Method | Path |
|--------|------|
| GET | `/api/v1/admin/appointments?status=&date=&page=&page_size=` |
| GET | `/api/v1/admin/appointments/:id` |

Blocking statuses for the calendar: `booked`, `paid`, `acknowledged`.
