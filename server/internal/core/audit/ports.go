package audit

import "context"

type Recorder interface {
	RecordEvent(ctx context.Context, input RecordEventInput) error
}

type Store interface {
	Recorder
	ListRecentEvents(ctx context.Context, limit int) ([]Event, error)
}
