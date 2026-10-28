package sessions_worker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
	"go.uber.org/zap"
)

const (
	testInterval = 10 * time.Millisecond
	testTimeout  = 2 * time.Second
)

// mockSessionsService — ручной мок SessionsService.
// Каждый вызов отправляет сигнал в канал calls, чтобы тесты
// ждали события, а не спали фиксированное время.
type mockSessionsService struct {
	calls     chan struct{}
	callCount atomic.Int32
	// errFn позволяет задать ошибку в зависимости от номера вызова (с 1).
	errFn   func(call int32) error
	lastCtx atomic.Value
}

func newMockSessionsService(errFn func(call int32) error) *mockSessionsService {
	return &mockSessionsService{
		calls: make(chan struct{}, 100),
		errFn: errFn,
	}
}

func (m *mockSessionsService) DeleteExpiredSession(ctx context.Context) error {
	n := m.callCount.Add(1)
	m.lastCtx.Store(ctx)

	select {
	case m.calls <- struct{}{}:
	default:
	}

	if m.errFn != nil {
		return m.errFn(n)
	}
	return nil
}

// waitCalls ждёт n вызовов сервиса или падает по таймауту.
func (m *mockSessionsService) waitCalls(t *testing.T, n int) {
	t.Helper()

	timeout := time.After(testTimeout)
	for i := 0; i < n; i++ {
		select {
		case <-m.calls:
		case <-timeout:
			t.Fatalf("таймаут: ожидали %d вызовов, получили %d", n, i)
		}
	}
}

func newTestLogger() *core_logger.Logger {
	logger, _ := zap.NewDevelopment()
	return &core_logger.Logger{Logger: logger}
}

func newTestCleaner(svc SessionsService, interval time.Duration) *SessionsCleaner {
	return NewSessionsCleaner(
		Config{CleanupInterval: interval},
		svc,
		newTestLogger(),
	)
}

// runCleaner запускает Run в горутине и возвращает канал с результатом.
func runCleaner(ctx context.Context, c *SessionsCleaner) <-chan error {
	done := make(chan error, 1)
	go func() {
		done <- c.Run(ctx)
	}()
	return done
}

// waitDone ждёт завершения Run и возвращает его ошибку.
func waitDone(t *testing.T, done <-chan error) error {
	t.Helper()

	select {
	case err := <-done:
		return err
	case <-time.After(testTimeout):
		t.Fatal("таймаут: Run не завершился после отмены контекста")
		return nil
	}
}

func TestNewSessionsCleaner(t *testing.T) {
	t.Run("конструктор сохраняет переданные зависимости", func(t *testing.T) {
		svc := newMockSessionsService(nil)
		logger := newTestLogger()
		cfg := Config{CleanupInterval: time.Minute}

		c := NewSessionsCleaner(cfg, svc, logger)

		require.NotNil(t, c)
		assert.Equal(t, cfg, c.config)
		assert.Same(t, svc, c.sessionsService)
		assert.Same(t, logger, c.log)
	})
}

func TestSessionsCleaner_Run(t *testing.T) {
	t.Run("завершается без ошибки при отмене контекста", func(t *testing.T) {
		svc := newMockSessionsService(nil)
		c := newTestCleaner(svc, time.Hour)

		ctx, cancel := context.WithCancel(context.Background())
		done := runCleaner(ctx, c)

		cancel()

		err := waitDone(t, done)
		assert.NoError(t, err)
		assert.Zero(t, svc.callCount.Load(), "до первого тика сервис не должен вызываться")
	})

	t.Run("уже отменённый контекст — немедленный выход", func(t *testing.T) {
		svc := newMockSessionsService(nil)
		c := newTestCleaner(svc, time.Hour)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := waitDone(t, runCleaner(ctx, c))
		assert.NoError(t, err)
		assert.Zero(t, svc.callCount.Load())
	})

	t.Run("вызывает DeleteExpiredSession на каждом тике", func(t *testing.T) {
		svc := newMockSessionsService(nil)
		c := newTestCleaner(svc, testInterval)

		ctx, cancel := context.WithCancel(context.Background())
		done := runCleaner(ctx, c)

		svc.waitCalls(t, 3)

		cancel()
		require.NoError(t, waitDone(t, done))
		assert.GreaterOrEqual(t, svc.callCount.Load(), int32(3))
	})

	t.Run("продолжает работу после ошибки сервиса", func(t *testing.T) {
		// Первые два вызова падают, дальше — успех.
		svc := newMockSessionsService(func(call int32) error {
			if call <= 2 {
				return errors.New("boom")
			}
			return nil
		})
		c := newTestCleaner(svc, testInterval)

		ctx, cancel := context.WithCancel(context.Background())
		done := runCleaner(ctx, c)

		// Если бы Run завершался на ошибке, вызовов после второго не было бы.
		svc.waitCalls(t, 4)

		cancel()
		err := waitDone(t, done)
		assert.NoError(t, err, "Run не должен возвращать ошибки сервиса")
	})

	t.Run("постоянные ошибки сервиса не останавливают воркер", func(t *testing.T) {
		svc := newMockSessionsService(func(int32) error {
			return errors.New("always fails")
		})
		c := newTestCleaner(svc, testInterval)

		ctx, cancel := context.WithCancel(context.Background())
		done := runCleaner(ctx, c)

		svc.waitCalls(t, 3)

		select {
		case err := <-done:
			t.Fatalf("Run неожиданно завершился: %v", err)
		default:
		}

		cancel()
		assert.NoError(t, waitDone(t, done))
	})

	t.Run("передаёт контекст в сервис", func(t *testing.T) {
		type ctxKey struct{}

		svc := newMockSessionsService(nil)
		c := newTestCleaner(svc, testInterval)

		ctx, cancel := context.WithCancel(
			context.WithValue(context.Background(), ctxKey{}, "marker"),
		)
		done := runCleaner(ctx, c)

		svc.waitCalls(t, 1)

		got, ok := svc.lastCtx.Load().(context.Context)
		require.True(t, ok, "сервис должен получить контекст")
		assert.Equal(t, "marker", got.Value(ctxKey{}))

		cancel()
		require.NoError(t, waitDone(t, done))
	})

	t.Run("после остановки сервис больше не вызывается", func(t *testing.T) {
		svc := newMockSessionsService(nil)
		c := newTestCleaner(svc, testInterval)

		ctx, cancel := context.WithCancel(context.Background())
		done := runCleaner(ctx, c)

		svc.waitCalls(t, 1)

		cancel()
		require.NoError(t, waitDone(t, done))

		after := svc.callCount.Load()
		time.Sleep(5 * testInterval)

		assert.Equal(t, after, svc.callCount.Load(), "вызовы после остановки недопустимы")
	})
}
