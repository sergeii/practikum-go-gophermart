package withdrawals

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/sergeii/practikum-go-gophermart/internal/core/users"
)

type Withdrawal struct {
	ID          int
	User        users.User
	Number      string
	Sum         decimal.Decimal
	ProcessedAt time.Time
}

var Blank Withdrawal // nolint: gochecknoglobals

func New(number string, userID int, sum decimal.Decimal) Withdrawal {
	return Withdrawal{
		User:        users.NewFromID(userID),
		Number:      number,
		Sum:         sum,
		ProcessedAt: time.Now(),
	}
}

func NewFromRepo(id int, number string, userID int, sum decimal.Decimal, processedAt time.Time) Withdrawal {
	return Withdrawal{
		ID:          id,
		User:        users.NewFromID(userID),
		Number:      number,
		Sum:         sum,
		ProcessedAt: processedAt,
	}
}
