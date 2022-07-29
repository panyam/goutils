package dal

import (
	"strings"
	"time"
)

type BaseEntity struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Resource struct {
	BaseEntity

	Id string

	// Owner/Creator of this resource
	UserId string

	// Visibility status
	Visibility string //  "public" | "private" | "limited";

	// Who can see this score
	VisibleTo []string

	Version string
}

func (res *Resource) IsVisibleTo(userId string) bool {
	return res.UserId == userId || res.Visibility == "public"
}

// And others things here
func (res *Resource) hasKey() bool {
	return strings.Trim(res.Id, " ") == ""
}
