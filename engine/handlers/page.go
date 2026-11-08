package handlers

import (
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/utils"
	"github.com/ebrgr/TuneTrail/internal/webquery"
	"net/http"
	"time"
)

// Additive metadata lets every UI display the interval actually queried.
func writePage[T any](w http.ResponseWriter, tf db.Timeframe, page *db.PaginatedResponse[T]) {
	from, to := db.TimeframeToTimeRange(tf)
	var start *string
	if !from.IsZero() {
		v := from.Format(time.RFC3339)
		start = &v
	}
	utils.WriteJSON(w, 200, struct {
		*db.PaginatedResponse[T]
		Range struct {
			From     *string `json:"from"`
			To       string  `json:"to"`
			Timezone string  `json:"timezone"`
			Label    string  `json:"label"`
		} `json:"range"`
	}{PaginatedResponse: page, Range: struct {
		From     *string `json:"from"`
		To       string  `json:"to"`
		Timezone string  `json:"timezone"`
		Label    string  `json:"label"`
	}{start, to.Format(time.RFC3339), tf.Timezone.String(), webquery.RangeLabel(tf)}})
}
