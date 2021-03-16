package message

import "context"

type Event struct {
	Message
}

func (e *Event) Complete() {
	if e.Message.flush != nil {
		e.Message.flush(e)
	}
}
func NewEvent(mtype, name string, flush Flush) *Event {
	return NewEventWithContext(nil, mtype, name, flush)
}

func NewEventWithContext(ctx context.Context, mtype, name string, flush Flush) *Event {
	return &Event{
		Message: NewMessageWithContext(ctx, mtype, name, flush),
	}
}
