package model

import "time"

type Delegation struct {
	ID        int       `db:"id"`
	TzktID    int64     `db:"tzkt_id"`
	Timestamp time.Time `db:"timestamp"`
	Amount    int64     `db:"amount"`
	Delegator string    `db:"delegator"`
	Level     int64     `db:"level"`
}
