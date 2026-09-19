# Hairstyles API (admin)

Protected with `Authorization: Bearer <token>`.

## CRUD

| Method | Path | Notes |
|--------|------|--------|
| POST | `/api/v1/admin/hairstyles` | Create |
| GET | `/api/v1/admin/hairstyles` | List (`q`, `active`, `page`, `page_size`) |
| GET | `/api/v1/admin/hairstyles/:id` | Get one |
| PUT | `/api/v1/admin/hairstyles/:id` | Update (orphans media removed from Supabase) |
| DELETE | `/api/v1/admin/hairstyles/:id` | Delete + remove media (blocked later if appointments remain) |

### Body

```json
{
  "name": "Low fade",
  "description": "Clean taper fade",
  "walkInPriceKobo": 500000,
  "homeServicePriceKobo": 800000,
  "durationMinutes": 45,
  "imageUrls": ["https://…/hairstyles/….jpg"],
  "videoUrl": "",
  "active": true
}
```

Rules: 1–3 `imageUrls`, duration ≥ 1, prices ≥ 0, name + description required.

## Uploads

| Method | Path | Limits |
|--------|------|--------|
| POST | `/api/v1/admin/uploads` | Images ≤ **500KB** (jpeg/png/webp/gif) |
| POST | `/api/v1/admin/uploads/video` | Video ≤ **5MB** (mp4/mov/mkv) |

Multipart field: `file`. Response: `{ "url": "…" }`. Objects land under `hairstyles/` in the configured bucket.
