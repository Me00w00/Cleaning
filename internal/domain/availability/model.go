package availability

import "time"

type Period struct {
	ID        int64
	StaffID   int64
	StartsAt  time.Time
	EndsAt    time.Time
	Reason    string
	CreatedAt time.Time
}
