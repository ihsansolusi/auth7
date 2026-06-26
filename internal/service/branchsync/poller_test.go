package branchsync

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// TestDeactivateAbsent_EmptySeenIsNoOp is the critical safety guard: a
// successful pass that reports zero branches must NOT touch the database,
// otherwise a misconfigured source or a transient empty response would wipe
// the whole projection. The nil pool proves no query is issued — if the guard
// regressed, pool.Exec would panic on the nil receiver.
func TestDeactivateAbsent_EmptySeenIsNoOp(t *testing.T) {
	p := &Poller{
		cfg:    Config{OrgID: uuid.MustParse("00000000-0000-0000-0000-000000000001")},
		pool:   nil, // any DB access here would panic — that's the assertion
		logger: zerolog.Nop(),
	}

	n, err := p.deactivateAbsent(context.Background(), map[uuid.UUID]struct{}{})
	if err != nil {
		t.Fatalf("expected nil error on empty seen set, got %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0 deactivations on empty seen set, got %d", n)
	}
}

// TestNewPoller_EnabledGate verifies the poller only arms when both SourceURL
// and ClientID are present.
func TestNewPoller_EnabledGate(t *testing.T) {
	cases := []struct {
		name      string
		sourceURL string
		clientID  string
		want      bool
	}{
		{"both set", "http://enterprise/branches", "branchsync", true},
		{"missing url", "", "branchsync", false},
		{"missing client", "http://enterprise/branches", "", false},
		{"both empty", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewPoller(Config{SourceURL: tc.sourceURL, ClientID: tc.clientID}, nil, zerolog.Nop())
			if p.enabled != tc.want {
				t.Fatalf("enabled = %v, want %v", p.enabled, tc.want)
			}
		})
	}
}
