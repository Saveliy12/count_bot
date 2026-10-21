package bot

import (
	"sync"
	"time"

	"github.com/Saveliy12/expenses/internal/model"
)

const (
	pendingDeletionTTL      = 2 * time.Minute
	pendingClarificationTTL = 5 * time.Minute
)

type pendingKind string

const (
	pendingKindDeletion      pendingKind = "deletion"
	pendingKindClarification pendingKind = "clarification"
)

type pendingState struct {
	kind         pendingKind
	expense      model.Expense
	originalText string
	question     string
	expiresAt    time.Time
}

type pendingStates struct {
	mu    sync.Mutex
	items map[int64]pendingState
}

func newPendingStates() *pendingStates {
	return &pendingStates{
		items: make(map[int64]pendingState),
	}
}

func (p *pendingStates) putDeletion(telegramID int64, expense model.Expense, now time.Time) {
	p.put(telegramID, pendingState{
		kind:      pendingKindDeletion,
		expense:   expense,
		expiresAt: now.Add(pendingDeletionTTL),
	})
}

func (p *pendingStates) putClarification(
	telegramID int64,
	originalText string,
	question string,
	now time.Time,
) {
	p.put(telegramID, pendingState{
		kind:         pendingKindClarification,
		originalText: originalText,
		question:     question,
		expiresAt:    now.Add(pendingClarificationTTL),
	})
}

func (p *pendingStates) take(telegramID int64, now time.Time) (pendingState, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	state, ok := p.items[telegramID]
	if !ok {
		return pendingState{}, false
	}
	delete(p.items, telegramID)

	if now.After(state.expiresAt) {
		return pendingState{}, false
	}

	return state, true
}

func (p *pendingStates) clear(telegramID int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.items, telegramID)
}

func (p *pendingStates) put(telegramID int64, state pendingState) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.items[telegramID] = state
}
