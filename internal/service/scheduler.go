package service

import (
	"context"
	"log"
	"time"
)

func StartScheduler(ctx context.Context, logger *log.Logger, svc *ClassroomService, loc *time.Location) {
	go func() {
		for {
			next := nextRun(time.Now().In(loc), loc)
			timer := time.NewTimer(time.Until(next))
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			runWithRetry(ctx, logger, svc)
		}
	}()
}

func runWithRetry(ctx context.Context, logger *log.Logger, svc *ClassroomService) {
	delays := []time.Duration{0, 5 * time.Minute, 10 * time.Minute, 15 * time.Minute}
	for attempt, delay := range delays {
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}

		err := svc.SyncAllToday(ctx, []int{0, 1})
		if err == nil {
			logger.Printf("classroom sync finished on attempt %d", attempt+1)
			return
		}
		logger.Printf("classroom sync failed on attempt %d: %v", attempt+1, err)
	}
}

func nextRun(now time.Time, loc *time.Location) time.Time {
	today := time.Date(now.Year(), now.Month(), now.Day(), 5, 30, 0, 0, loc)
	if now.Before(today) {
		return today
	}
	return today.Add(24 * time.Hour)
}
