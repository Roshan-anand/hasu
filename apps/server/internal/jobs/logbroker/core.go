package logbroker

import (
	"context"
	"fmt"
	"sync"

	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/lib/database"
	"github.com/Roshan-anand/godploy/internal/lib/sse"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type PubData struct {
	ID  uuid.UUID
	Msg string
}

type EndLogData struct {
	DeploymentID uuid.UUID
	Status       types.DeploymentStatus
	Message      string
}

type Subscriber struct {
	SSE          *sse.SSE
	DeploymentID uuid.UUID
	IsNew        bool
}

type LogBuffer map[uuid.UUID][]string

type LogBrokerService struct {
	mu          sync.Mutex
	badgerDB    *database.BadgerDB
	quries      *db.Queries
	eg          *errgroup.Group
	egCtx       context.Context
	cancel      context.CancelFunc
	pubData     chan *PubData
	endData     chan *EndLogData
	subscribers map[uuid.UUID]*Subscriber
	logBuffer   LogBuffer
}

func NewLogBrokerService(q *db.Queries, db *database.BadgerDB) *LogBrokerService {
	pub := make(chan *PubData, 100)
	end := make(chan *EndLogData, 100)
	sub := make(map[uuid.UUID]*Subscriber, 100)
	logBuffer := make(LogBuffer)

	return &LogBrokerService{
		pubData:     pub,
		endData:     end,
		subscribers: sub,
		badgerDB:    db,
		quries:      q,
		logBuffer:   logBuffer,
	}
}

func (l *LogBrokerService) Start(ctx context.Context) {
	l.egCtx, l.cancel = context.WithCancel(ctx)
	l.eg, _ = errgroup.WithContext(l.egCtx)

	l.eg.Go(func() error {
		return l.worker(l.egCtx)
	})

}

func (l *LogBrokerService) Stop(ctx context.Context) error {
	close(l.pubData)
	close(l.endData)

	done := make(chan error, 1)
	go func() {
		done <- l.eg.Wait()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		l.cancel()
		return ctx.Err()
	}
}

func (l *LogBrokerService) worker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("LogBrokerService worker received shutdown signal, exiting...")
			return ctx.Err()

		case p, ok := <-l.pubData:
			if !ok {
				return fmt.Errorf("pubData channel closed, worker exiting")
			}
			l.publisher(p)

		case e, ok := <-l.endData:
			if !ok {
				return fmt.Errorf("pubData channel closed, worker exiting")
			}
			l.ender(e)
		}
	}
}

func (l *LogBrokerService) PublishLog(data *PubData) error {
	select {
	case <-l.egCtx.Done():
		return fmt.Errorf("cannot publish log, service is shutting down")
	case l.pubData <- data:
		return nil
	}
}

func (l *LogBrokerService) EndLogs(data *EndLogData) error {
	select {
	case <-l.egCtx.Done():
		return fmt.Errorf("cannot publish log, service is shutting down")
	case l.endData <- data:
		return nil
	}
}

// subscribe to logs of the deployment
func (l *LogBrokerService) Subscribe(userID uuid.UUID, sub *Subscriber) {
	l.mu.Lock()
	defer l.mu.Unlock()

	sub.IsNew = true
	l.subscribers[userID] = sub
}

// unsubscribe user from logs of the deployment
func (l *LogBrokerService) Unsubscribe(userID uuid.UUID) {
	l.mu.Lock()
	delete(l.subscribers, userID)
	l.mu.Unlock()
}
