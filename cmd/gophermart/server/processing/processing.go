package processing

import (
	"context"
	"errors"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/sergeii/practikum-go-gophermart/internal/application"
)

var ErrProcessingInterrupted = errors.New("processing is interrupted")

func Run(ctx context.Context, app *application.App, wg *sync.WaitGroup, failure chan error) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			// shutting down
			log.Info().Msg("Stopping processing of accrual queue")
			failure <- ErrProcessingInterrupted
			return
		default:
			wait := app.OrderService.ProcessNextOrder(ctx)
			<-wait
		}
	}
}
