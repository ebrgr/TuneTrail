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
	Logout  bool
}

// newSubmission bounds each session and replaces only its oldest ticket, preserving other sessions.
func (h *Handler) newSubmission(session uuid.UUID) (string, error) {
	return h.issueSubmission(session, false)
}

// issueSubmission renews a logout-only ticket so each page gets a full TTL without evicting edit forms.
func (h *Handler) issueSubmission(session uuid.UUID, logout bool) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	count := 0
	var oldest, reusableLogout uuid.UUID
	var oldestExpiry time.Time
	for k, v := range h.submissions {
		if now.After(v.Expires) {
			delete(h.submissions, k)
			continue
		}
		if v.Session == session && v.Logout == logout {
			if logout {
				reusableLogout = k
				continue
			}
			if count == 0 || v.Expires.Before(oldestExpiry) {
				oldest, oldestExpiry = k, v.Expires
			}
			count++
		}
	}
	if reusableLogout != uuid.Nil {
		h.submissions[reusableLogout] = submissionTicket{session, now.Add(30 * time.Minute), true}
		return reusableLogout.String(), nil
	}
	// Edit tickets keep their original expiry, so the earliest expiry identifies the oldest issue.
	if count >= maxSessionSubmissions {
		delete(h.submissions, oldest)
	}
	if len(h.submissions) >= 4096 {
		return "", db.ErrConflict
	}
	token, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	h.submissions[token] = submissionTicket{session, now.Add(30 * time.Minute), logout}
	return token.String(), nil
}

// consumeSubmission prevents duplicate mutations even when JavaScript is unavailable.
func (h *Handler) consumeSubmission(session uuid.UUID, raw string) error {
	return h.consumeTicket(session, raw, false)
}

// consumeTicket permits existing page tickets for logout and reserves logout tickets for logout.
func (h *Handler) consumeTicket(session uuid.UUID, raw string, logout bool) error {
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
	if !ok || ticket.Session != session || ticket.Logout && !logout || time.Now().After(ticket.Expires) {
		return db.ErrConflict
	}
	delete(h.submissions, token)
	return nil
}
