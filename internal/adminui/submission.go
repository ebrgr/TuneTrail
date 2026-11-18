package adminui

import (
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/google/uuid"
	"time"
)

const maxSessionSubmissions = 32

type submissionTicket struct {
	Session uuid.UUID
	Expires time.Time
}

// newSubmission bounds each session and replaces only its oldest ticket, preserving other sessions.
func (h *Handler) newSubmission(session uuid.UUID) (string, error) {
	token, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	count := 0
	var oldest uuid.UUID
	var oldestExpiry time.Time
	for k, v := range h.submissions {
		if now.After(v.Expires) {
			delete(h.submissions, k)
			continue
		}
		if v.Session == session {
			if count == 0 || v.Expires.Before(oldestExpiry) {
				oldest, oldestExpiry = k, v.Expires
			}
			count++
		}
	}
	// All tickets have the same TTL, so the earliest expiry identifies the oldest issue.
	if count >= maxSessionSubmissions {
		delete(h.submissions, oldest)
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
