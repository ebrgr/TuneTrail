// Package dedup classifies shadow receipts without changing listening history.
package dedup

import (
	"strings"
	"time"

	"github.com/ebrgr/TuneTrail/internal/ingest"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

type Policy struct {
	Version string
	Window  time.Duration
}

var Shadow = Policy{Version: "shadow-v2", Window: 2 * time.Second}

type Receipt struct {
	ID, Source, Namespace, ExternalID, Digest, BatchDigest, Precision, State, Mode, Decision, DigestVersion string
	Snapshot                                                                                                ingest.Snapshot
	Time                                                                                                    time.Time
}

type Evidence struct {
	Candidate          string `json:"candidate_receipt_id,omitempty"`
	DeltaSeconds       int64  `json:"delta_seconds,omitempty"`
	Precision          string `json:"precision"`
	RecordingMatch     bool   `json:"recording_mbid_match,omitempty"`
	CandidatePrecision string `json:"candidate_precision,omitempty"`
}

type Decision struct {
	Value, Reason, RuleVersion string
	Evidence                   Evidence
}

// Compare treats a trusted key as identity evidence and time as candidate selection only.
func Compare(policy Policy, receipt Receipt, candidate *Receipt, sameKey bool) Decision {
	result := Decision{Value: "different", Reason: "no_candidate", RuleVersion: policy.Version, Evidence: Evidence{Precision: receipt.Precision}}
	if candidate == nil {
		return result
	}
	result.Evidence.Candidate = candidate.ID
	result.Evidence.DeltaSeconds = int64(receipt.Time.Sub(candidate.Time) / time.Second)
	result.Evidence.CandidatePrecision = candidate.Precision
	if sameKey {
		if receipt.Digest == candidate.Digest && receipt.BatchDigest == candidate.BatchDigest {
			result.Value, result.Reason = "same", "trusted_key_equivalent"
		} else {
			result.Value, result.Reason = "review", "trusted_key_conflict"
		}
	} else if receipt.Source == candidate.Source && receipt.Namespace == candidate.Namespace &&
		receipt.ExternalID != "" && candidate.ExternalID != "" && receipt.ExternalID != candidate.ExternalID {
		result.Reason = "distinct_external_ids"
	} else if receipt.Time.Sub(candidate.Time).Abs() <= policy.Window {
		a, b := receipt.Snapshot, candidate.Snapshot
		result.Evidence.RecordingMatch = sameRecording(a, b)
		if result.Evidence.RecordingMatch ||
			(NormalizeName(a.Artist) == NormalizeName(b.Artist) && NormalizeName(a.Title) == NormalizeName(b.Title)) {
			result.Value, result.Reason = "review", "ambiguous_time_candidate"
		}
	}
	return result
}

// NormalizeName folds Unicode case, canonical composition and runs of whitespace.
func NormalizeName(name string) string {
	return norm.NFC.String(cases.Fold().String(strings.Join(strings.Fields(name), " ")))
}

// sameRecording compares declared and mapped MusicBrainz recording identifiers.
func sameRecording(a, b ingest.Snapshot) bool {
	for _, left := range []string{a.Recording, a.MappedRecording} {
		if left != "" && (strings.EqualFold(left, b.Recording) || strings.EqualFold(left, b.MappedRecording)) {
			return true
		}
	}
	return false
}
