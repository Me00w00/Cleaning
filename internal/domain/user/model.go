package user

import "time"

type Role string

const (
	RoleAdmin   Role = "admin"
	RoleManager Role = "manager"
	RoleStaff   Role = "staff"
	RoleClient  Role = "client"
)

type User struct {
	ID           int64
	Login        string
	PasswordHash string
	Role         Role
	FullName     string
	Phone        string
	Email        string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (u User) RoleLabel() string {
	switch u.Role {
	case RoleAdmin:
		return "Administrator"
	case RoleManager:
		return "Manager"
	case RoleStaff:
		return "Staff"
	case RoleClient:
		return "Client"
	default:
		return string(u.Role)
	}
}
