package transactor

import "context"

type Transactor interface {
	WithTransaction(context.Context, func(ctx context.Context) error) error
}
