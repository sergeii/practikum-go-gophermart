package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Withdrawal struct {
	ID          int
	User        User
	Number      string
	Sum         decimal.Decimal
	ProcessedAt time.Time
}

func NewCandidateWithdrawal(number string, userID int, sum decimal.Decimal) Withdrawal {
	return Withdrawal{
		User:        User{ID: userID},
		Number:      number,
		Sum:         sum,
		ProcessedAt: time.Now(),
	}
}

func NewAcceptedWithdrawal(id int, number string, userID int, sum decimal.Decimal, processedAt time.Time) Withdrawal {
	return Withdrawal{
		ID:          id,
		User:        User{ID: userID},
		Number:      number,
		Sum:         sum,
		ProcessedAt: processedAt,
	}
}
