package registry

import (
	"context"
	"log"
	"math"
	"math/rand"
	"time"

	"evil-rkn/internal/domain"
)

type Fetcher interface {
	FetchRegistry(ctx context.Context) (*domain.Registry, error)
}

type Config struct {
	Interval       time.Duration // base update interval
	InitialBackoff time.Duration // initial backoff delay
	MaxBackoff     time.Duration // maximum backoff delay
}

// Start runs background registry updates until the context stops.
func Start(ctx context.Context, cfg Config, src Fetcher, holder *Holder) error {
	if cfg.Interval <= 0 {
		return nil // config should already be validated
	}
	if cfg.InitialBackoff <= 0 {
		cfg.InitialBackoff = 30 * time.Second
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = 30 * time.Minute
	}

	// Perform the first update immediately on startup
	if err := updateOnce(ctx, src, holder); err != nil {
		log.Printf("registry: initial update failed: %v", err)
	} else {
		log.Printf("registry: initial update succeeded")
	}

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	var consecutiveFailures int

	for {
		select {
		case <-ctx.Done():
			log.Printf("registry: updater stopped: %v", ctx.Err())
			return ctx.Err()

		case <-ticker.C:
			if err := updateOnce(ctx, src, holder); err != nil {
				consecutiveFailures++
				backoff := calcBackoff(cfg.InitialBackoff, cfg.MaxBackoff, consecutiveFailures)

				log.Printf("registry: update failed (attempt #%d), backoff=%s: %v",
					consecutiveFailures, backoff, err)

				timer := time.NewTimer(backoff)
				select {
				case <-ctx.Done():
					timer.Stop()
					log.Printf("registry: updater stopped during backoff: %v", ctx.Err())
					return ctx.Err()
				case <-timer.C:
				}
				continue
			}

			if consecutiveFailures > 0 {
				log.Printf("registry: update recovered after %d failures", consecutiveFailures)
			}
			consecutiveFailures = 0
		}
	}
}

func calcBackoff(initial, max time.Duration, failures int) time.Duration {
	pow := math.Pow(2, float64(failures-1))
	backoff := time.Duration(float64(initial) * pow)
	if backoff > max {
		backoff = max
	}

	// Add jitter to avoid synchronized retries
	jitterFrac := 0.2
	jitter := time.Duration(rand.Float64()*2*jitterFrac*float64(backoff)) -
		time.Duration(jitterFrac*float64(backoff))

	return backoff + jitter
}

// updateOnce fetches the registry and updates the holder.
func updateOnce(ctx context.Context, src Fetcher, holder *Holder) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	reg, err := src.FetchRegistry(ctx)
	if err != nil {
		return err
	}

	holder.Set(reg)
	return nil
}
