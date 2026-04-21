ALTER TABLE staff_unavailability ADD COLUMN starts_at TEXT;
ALTER TABLE staff_unavailability ADD COLUMN ends_at TEXT;

UPDATE staff_unavailability
SET starts_at = COALESCE(starts_at, date(date_from) || ' 00:00'),
    ends_at = COALESCE(ends_at, date(date_to) || ' 23:59');

CREATE INDEX IF NOT EXISTS idx_staff_unavailability_period_bounds
ON staff_unavailability(staff_id, starts_at, ends_at);
