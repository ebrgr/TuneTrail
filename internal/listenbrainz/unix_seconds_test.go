package listenbrainz

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnixSecondsAcceptsIntegerAndDecimal(t *testing.T) {
	for in, want := range map[string]UnixSeconds{
		`1700000000`:     1700000000,
		`1700000000.0`:   1700000000,
		`1700000000.987`: 1700000000,
		`1.7e9`:          1700000000,
		`null`:           0,
	} {
		var p LbzSubmitListenPayload
		require.NoError(t, json.Unmarshal([]byte(`{"listened_at":`+in+`,"track_metadata":{}}`), &p), in)
		require.Equal(t, want, p.ListenedAt, in)
	}
}

func TestUnixSecondsRejectsInvalid(t *testing.T) {
	for _, in := range []string{`"1700000000"`, `true`, `1e400`} {
		var p LbzSubmitListenPayload
		require.Error(t, json.Unmarshal([]byte(`{"listened_at":`+in+`}`), &p), in)
	}
}

func TestUnixSecondsMarshalsAsInteger(t *testing.T) {
	b, err := json.Marshal(LbzSubmitListenPayload{ListenedAt: 1700000000})
	require.NoError(t, err)
	require.Contains(t, string(b), `"listened_at":1700000000`)
}
