package db_test

import (
	"testing"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/stretchr/testify/require"
)

func TestPeriodUnset(t *testing.T) {
	var p db.Period
	require.True(t, p.IsZero())
}
