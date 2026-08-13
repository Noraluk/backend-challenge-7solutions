package application

import (
	"context"
	"log/slog"
	"time"

	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
)

func RunUserCountWorker(ctx context.Context, repository ports.UserRepository, logger *slog.Logger) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	runUserCountWorker(ctx, repository, logger, ticker.C)
}

func runUserCountWorker(ctx context.Context, repository ports.UserRepository, logger *slog.Logger, ticks <-chan time.Time) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
			count, err := repository.Count(ctx)
			if err != nil {
				logger.Error("count users", slog.Any("error", err))
				continue
			}
			logger.Info("user count", slog.Int64("count", count))
		}
	}
}
