// Package dedup classifies shadow receipts without changing listening history.
package dedup

import (
	"strings"
	"time"

	"github.com/ebrgr/TuneTrail/internal/ingest"
)

type Policy struct {
	Version string
	Window  time.Duration
}

var Shadow = Policy{Version: "shadow-v1", Window: 2 * time.Second}

type Receipt struct {
	ID, Source, Namespace, ExternalID, Digest, BatchDigest, Precision, State string
	Snapshot                                                                 ingest.Snapshot
	Time                                                                     time.Time
}

type Evidence struct {
	Candidate          string `json:"candidate_receipt_id,omitempty"`
	DeltaSeconds       int64  `json:"delta_seconds,omitempty"`
	Precision          string `json:"precision"`
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
	} else if receipt.Time.Sub(candidate.Time).Abs() <= policy.Window &&
		((receipt.Snapshot.Recording != "" && receipt.Snapshot.Recording == candidate.Snapshot.Recording) ||
			(strings.EqualFold(receipt.Snapshot.Artist, candidate.Snapshot.Artist) && strings.EqualFold(receipt.Snapshot.Title, candidate.Snapshot.Title))) {
		result.Value, result.Reason = "review", "ambiguous_time_candidate"
	}
	return result
}
