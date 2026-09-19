# End-to-end QA checklist (Phase 9)

Run with API (`air` / `go run .`), admin (`:3001`), and client (`:3000`) up. SMTP + Paystack test keys recommended so emails/payments are real.

## 1. Styles

- [ ] Admin creates **2 styles** (one active, one inactive); inactive has images.
- [ ] Client catalogue shows **only active**; search filters name/description.
- [ ] Style detail: gallery, walk-in vs home prices, slots reload on service/day change.

## 2. Walk-in happy path

- [ ] Book walk-in → Paystack success → `/book/success` shows tracking `HBL-…`.
- [ ] Customer + admin **paid** emails.
- [ ] `/track` with tracking + email shows timeline.
- [ ] Admin: paid → acknowledge → completed; **completed** email to customer only.

## 3. Home service + missed + reschedule

- [ ] Book home service **with address** → pay → tracking.
- [ ] Admin: paid → acknowledge → **missed**; customer missed email with reschedule CTA.
- [ ] Customer reschedules once (free) → status `paid`; customer + admin emails; old slot free.
- [ ] Admin re-ack → complete.

## 4. Holds & conflicts

- [ ] Create unpaid `booked` hold; after **15m** → `abandoned`; slot bookable again.
- [ ] Two clients race same slot → one succeeds, other **409** `slot_unavailable`.
- [ ] Stale slot on book form → soft-refresh slots (`invalid_start_time`).

## 5. Delete guards

- [ ] Delete style with future `paid`/`missed` appointment → **409** blocked message in admin.
- [ ] After only `completed`/`abandoned` (or none), delete succeeds and Supabase media is gone.

## 6. Jobs (logs)

- [ ] Startup logs: abandon hold, appointment reminder, post-time nag.
- [ ] Optional: paid start in ~15m (created ≥15m earlier) → one reminder pair; end in past → admin nag every ~30m until completed/missed.
- [ ] Book starting in ~5m → no T−15 reminder.

## 7. Content / polish

- [ ] `/contact` and `/terms` render (placeholders OK until real details).
- [ ] Terms mention non-refundable pay + free missed reschedule.
- [ ] Empty catalogue / empty appointments / track not-found are clear, not blank crashes.
