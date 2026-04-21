package order

import "time"

type Status string

type PaymentStatus string

const (
	StatusNew             Status = "new"
	StatusAssignedManager Status = "assigned_manager"
	StatusAssignedStaff   Status = "assigned_staff"
	StatusStaffConfirmed  Status = "staff_confirmed"
	StatusInProgress      Status = "in_progress"
	StatusCompleted       Status = "completed"
	StatusClosed          Status = "closed"
	StatusCancelled       Status = "cancelled"
)

const (
	PaymentStatusUnpaid PaymentStatus = "unpaid"
	PaymentStatusPaid   PaymentStatus = "paid"
)

type Address struct {
	ID       int64
	City     string
	Street   string
	House    string
	Floor    string
	Flat     string
	Entrance string
	Comment  string
}

type Contact struct {
	ID       int64
	FullName string
	Phone    string
	Email    string
}

type StatusHistoryEntry struct {
	ID              int64
	OrderID         int64
	OldStatus       Status
	NewStatus       Status
	ChangedByUserID int64
	ChangedByName   string
	ChangedAt       time.Time
	Comment         string
}

type ServiceCatalogItem struct {
	ID                  int64
	Code                string
	Name                string
	BasePrice           int
	PricePerSquareMeter int
	PricePerWindow      int
	IsActive            bool
}

type Order struct {
	ID                int64
	ClientID          int64
	ManagerID         int64
	StaffID           int64
	Client            Contact
	Manager           Contact
	Staff             Contact
	Address           Address
	ScheduledDate     time.Time
	ScheduledTimeFrom string
	ScheduledTimeTo   string
	ServiceType       string
	Details           string
	Square            int
	WindowCount       int
	Status            Status
	PaymentStatus     PaymentStatus
	PriceTotal        int
	CancelReason      string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
