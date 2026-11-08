package adminui

import (
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/google/uuid"
	"time"
)

type submissionTicket struct {
	Session uuid.UUID
	Expires time.Time
}

// newSubmission binds a one-use form ticket to a session, preserving independent tabs.
func (h *Handler) newSubmission(session uuid.UUID) (string, error) {
	token, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	for k, v := range h.submissions {
		if now.After(v.Expires) {
			delete(h.submissions, k)
		}
	}
	if len(h.submissions) >= 4096 {
		return "", db.ErrConflict
	}
	h.submissions[token] = submissionTicket{session, now.Add(30 * time.Minute)}
	return token.String(), nil
}

// consumeSubmission prevents duplicate mutations even when JavaScript is unavailable.
func (h *Handler) consumeSubmission(session uuid.UUID, raw string) error {
	if raw == "" && !h.requireSubmission {
		return nil
	}
	token, err := uuid.Parse(raw)
	if err != nil {
		return db.ErrConflict
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	ticket, ok := h.submissions[token]
	if !ok || ticket.Session != session || time.Now().After(ticket.Expires) {
		return db.ErrConflict
	}
	delete(h.submissions, token)
	return nil
}
