package filtering

import "time"

type Rule struct {
	ID         string
	Name       string
	Expression string // CEL expression that must evaluate to bool
	Priority   int
	Enabled    bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
