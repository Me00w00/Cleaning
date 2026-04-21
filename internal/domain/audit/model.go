package audit

import "time"

type Entry struct {
	ID          int64
	EntityType  string
	EntityID    int64
	Action      string
	ActorUserID int64
	ActorName   string
	PayloadJSON string
	CreatedAt   time.Time
}
