package webui

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/engine/handlers"
	"github.com/ebrgr/TuneTrail/internal/db"
)

// TestHistoryHTMLAndJSONShareCivilMonthBounds checks an independent Kathmandu fixture.
func TestHistoryHTMLAndJSONShareCivilMonthBounds(t *testing.T) {
	s := pageStore(t)
	loc, err := time.LoadLocation("Asia/Kathmandu")
	if err != nil {
		t.Fatal(err)
	}
	for _, at := range []time.Time{time.Date(2024, 5, 1, 0, 0, 0, 0, loc), time.Date(2024, 5, 31, 23, 59, 59, 0, loc)} {
		if err = s.SaveListen(context.Background(), db.SaveListenOpts{ProfileID: 1, TrackID: 2, Time: at}); err != nil {
			t.Fatal(err)
		}
	}
	h, err := NewHandler(s, Options{})
	if err != nil {
		t.Fatal(err)
	}
	query := "?year=2024&month=5&tz=Asia%2FKathmandu"
	html := pageRequest(h, "/alpha/listens"+query)
	if html.Code != 200 || strings.Count(html.Body.String(), `data-track-id="2"`) != 2 {
		t.Fatalf("HTML failed to include both edge listens: status=%d", html.Code)
	}
	req := httptest.NewRequest("GET", "/"+query, nil)
	req = req.WithContext(db.WithProfile(req.Context(), 1))
	w := httptest.NewRecorder()
	handlers.GetListensHandler(s)(w, req)
	var page struct {
		Total int `json:"total_record_count"`
	}
	if err = json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || page.Total != 2 {
		t.Fatalf("JSON status=%d count=%d", w.Code, page.Total)
	}
}
