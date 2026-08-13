package application

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/Noraluk/backend-challenge-7solutions/internal/mocks"
	"go.uber.org/mock/gomock"
)

func TestRunUserCountWorkerLogsCountOnTick(t *testing.T) {
	controller := gomock.NewController(t)
	repository := mocks.NewMockUserRepository(controller)
	counted := make(chan struct{})
	repository.EXPECT().Count(gomock.Any()).DoAndReturn(func(context.Context) (int64, error) {
		close(counted)
		return 42, nil
	})
	var output bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&output, nil))
	ctx, cancel := context.WithCancel(context.Background())
	ticks := make(chan time.Time)
	done := make(chan struct{})
	go func() {
		runUserCountWorker(ctx, repository, logger, ticks)
		close(done)
	}()

	ticks <- time.Now()
	<-counted
	cancel()
	<-done

	if logEntry := output.String(); !strings.Contains(logEntry, `msg="user count"`) || !strings.Contains(logEntry, "count=42") {
		t.Errorf("log entry = %q, want user count 42", logEntry)
	}
}

func TestRunUserCountWorkerContinuesAfterRepositoryError(t *testing.T) {
	controller := gomock.NewController(t)
	repository := mocks.NewMockUserRepository(controller)
	firstCall := make(chan struct{})
	secondCall := make(chan struct{})
	databaseError := errors.New("database failed")
	gomock.InOrder(
		repository.EXPECT().Count(gomock.Any()).DoAndReturn(func(context.Context) (int64, error) {
			close(firstCall)
			return 0, databaseError
		}),
		repository.EXPECT().Count(gomock.Any()).DoAndReturn(func(context.Context) (int64, error) {
			close(secondCall)
			return 7, nil
		}),
	)
	var output bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&output, nil))
	ctx, cancel := context.WithCancel(context.Background())
	ticks := make(chan time.Time)
	done := make(chan struct{})
	go func() {
		runUserCountWorker(ctx, repository, logger, ticks)
		close(done)
	}()

	ticks <- time.Now()
	<-firstCall
	ticks <- time.Now()
	<-secondCall
	cancel()
	<-done

	logEntries := output.String()
	if !strings.Contains(logEntries, "database failed") || !strings.Contains(logEntries, "count=7") {
		t.Errorf("log entries = %q, want error followed by count", logEntries)
	}
}

func TestRunUserCountWorkerStopsWhenContextIsCancelled(t *testing.T) {
	controller := gomock.NewController(t)
	repository := mocks.NewMockUserRepository(controller)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		runUserCountWorker(ctx, repository, slog.Default(), make(chan time.Time))
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop after context cancellation")
	}
}
