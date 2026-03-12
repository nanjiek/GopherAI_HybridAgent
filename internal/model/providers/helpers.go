package providers

import (
	"context"
	"errors"
	"time"
)

func withRetry(ctx context.Context, attempts int, baseDelay time.Duration, fn func(context.Context) error) error {
	var err error
	for i := 0; i < attempts; i++ {
		callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err = fn(callCtx)
		cancel()
		if err == nil {
			return nil
		}
		if i == attempts-1 {
			break
		}
		delay := baseDelay * time.Duration(i+1)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return err
}

func splitToTokens(text string) []string {
	if text == "" {
		return nil
	}
	out := make([]string, 0, len(text))
	for _, r := range text {
		out = append(out, string(r))
	}
	return out
}

var errCircuitOpen = errors.New("provider circuit breaker open")
