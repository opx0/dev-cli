package pipeline

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestEventBus_Subscribe(t *testing.T) {
	bus := NewEventBus()
	received := false

	bus.Subscribe(EventCommandStart, func(e Event) {
		received = true
	})

	bus.Publish(Event{
		Type:      EventCommandStart,
		Timestamp: time.Now(),
		Source:    "test",
	})

	if !received {
		t.Error("handler should have received the event")
	}
}

func TestEventBus_SubscribeAll(t *testing.T) {
	bus := NewEventBus()
	count := 0

	bus.SubscribeAll(func(e Event) {
		count++
	})

	bus.Publish(Event{Type: EventCommandStart})
	bus.Publish(Event{Type: EventCommandComplete})
	bus.Publish(Event{Type: EventContainerLog})

	if count != 3 {
		t.Errorf("expected 3 events, got %d", count)
	}
}

func TestEventBus_Publish_ToCorrectHandlers(t *testing.T) {
	bus := NewEventBus()
	startCount := 0
	completeCount := 0

	bus.Subscribe(EventCommandStart, func(e Event) {
		startCount++
	})
	bus.Subscribe(EventCommandComplete, func(e Event) {
		completeCount++
	})

	bus.Publish(Event{Type: EventCommandStart})
	bus.Publish(Event{Type: EventCommandStart})
	bus.Publish(Event{Type: EventCommandComplete})

	if startCount != 2 {
		t.Errorf("expected 2 start events, got %d", startCount)
	}
	if completeCount != 1 {
		t.Errorf("expected 1 complete event, got %d", completeCount)
	}
}

func TestEventBus_PublishMultipleHandlers(t *testing.T) {
	bus := NewEventBus()
	handler1Called := false
	handler2Called := false

	bus.Subscribe(EventCommandError, func(e Event) {
		handler1Called = true
	})
	bus.Subscribe(EventCommandError, func(e Event) {
		handler2Called = true
	})

	bus.Publish(Event{Type: EventCommandError})

	if !handler1Called || !handler2Called {
		t.Error("both handlers should be called")
	}
}

func TestEventBus_RecentEvents(t *testing.T) {
	bus := NewEventBus()

	for i := 0; i < 5; i++ {
		bus.Publish(Event{
			Type:    EventCommandOutput,
			BlockID: string(rune('a' + i)),
		})
	}

	recent := bus.RecentEvents(3)
	if len(recent) != 3 {
		t.Errorf("expected 3 recent events, got %d", len(recent))
	}

	if recent[0].BlockID != "c" || recent[2].BlockID != "e" {
		t.Error("should return most recent events in order")
	}
}

func TestEventBus_RecentByType(t *testing.T) {
	bus := NewEventBus()

	bus.Publish(Event{Type: EventCommandStart, BlockID: "1"})
	bus.Publish(Event{Type: EventCommandError, BlockID: "2"})
	bus.Publish(Event{Type: EventCommandStart, BlockID: "3"})
	bus.Publish(Event{Type: EventCommandComplete, BlockID: "4"})

	results := bus.RecentByType(EventCommandStart, 10)
	if len(results) != 2 {
		t.Errorf("expected 2 start events, got %d", len(results))
	}
}

func TestEventBus_HistoryLimit(t *testing.T) {
	bus := NewEventBus()
	bus.maxHistory = 5

	for i := 0; i < 10; i++ {
		bus.Publish(Event{Type: EventCommandOutput, BlockID: string(rune('0' + i))})
	}

	all := bus.RecentEvents(100)
	if len(all) != 5 {
		t.Errorf("expected 5 events (maxHistory), got %d", len(all))
	}

	if all[0].BlockID != "5" {
		t.Errorf("oldest event should be '5', got '%s'", all[0].BlockID)
	}
}

func TestEventBus_ConcurrentPublish(t *testing.T) {
	bus := NewEventBus()
	var count int64
	var wg sync.WaitGroup

	bus.SubscribeAll(func(e Event) {
		atomic.AddInt64(&count, 1)
	})

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Publish(Event{Type: EventSystemStats})
		}()
	}

	wg.Wait()

	if count != 100 {
		t.Errorf("expected 100 events handled, got %d", count)
	}
}

func TestEventBus_EventData(t *testing.T) {
	bus := NewEventBus()
	var receivedData interface{}

	bus.Subscribe(EventAISuggestion, func(e Event) {
		receivedData = e.Data
	})

	testData := map[string]string{"suggestion": "try npm install"}
	bus.Publish(Event{
		Type: EventAISuggestion,
		Data: testData,
	})

	if receivedData == nil {
		t.Error("event data should be passed to handler")
	}

	data, ok := receivedData.(map[string]string)
	if !ok || data["suggestion"] != "try npm install" {
		t.Error("event data should match what was published")
	}
}

func TestEventBus_Timestamp(t *testing.T) {
	bus := NewEventBus()

	before := time.Now()
	bus.Publish(Event{
		Type:      EventSystemAlert,
		Timestamp: time.Now(),
	})
	after := time.Now()

	recent := bus.RecentEvents(1)
	if len(recent) != 1 {
		t.Fatal("expected 1 event")
	}

	ts := recent[0].Timestamp
	if ts.Before(before) || ts.After(after) {
		t.Error("event timestamp should be preserved")
	}
}
