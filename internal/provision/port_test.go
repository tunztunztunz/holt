package provision

import (
	"testing"

	"github.com/tunztunztunz/holt/internal/config"
)

func TestAllocatePort(t *testing.T) {
	t.Run("nil block means no port", func(t *testing.T) {
		got, err := AllocatePort(nil, "site", nil)
		if got != 0 || err != nil {
			t.Errorf("AllocatePort(nil) = (%d, %v), want (0, nil)", got, err)
		}
	})

	t.Run("unknown strategy errors", func(t *testing.T) {
		p := &config.PortBlock{Range: [2]int{4000, 4999}, Strategy: "weird"}
		if _, err := AllocatePort(p, "site", nil); err == nil {
			t.Fatal("want error for unknown strategy, got nil")
		}
	})

	t.Run("a fully-taken range is exhausted", func(t *testing.T) {
		// Every port marked taken short-circuits before isFree, so this is
		// deterministic and never touches a real socket.
		p := &config.PortBlock{Range: [2]int{4000, 4001}, Strategy: config.PortFree}
		taken := map[int]bool{4000: true, 4001: true}
		if _, err := AllocatePort(p, "site", taken); err == nil {
			t.Fatal("want exhaustion error, got nil")
		}
	})

	// The success paths probe real sockets via isFree, so we assert only that the
	// result is within range and not in `taken` — not an exact port. The wide
	// range makes finding a free port near-certain.
	t.Run("free strategy skips taken ports", func(t *testing.T) {
		p := &config.PortBlock{Range: [2]int{41000, 41099}, Strategy: config.PortFree}
		taken := map[int]bool{41000: true}
		assertAllocated(t, p, "site", taken)
	})

	t.Run("hash strategy returns a port in range", func(t *testing.T) {
		p := &config.PortBlock{Range: [2]int{41000, 41099}, Strategy: config.PortHash}
		assertAllocated(t, p, "site", nil)
	})
}

// assertAllocated runs AllocatePort and checks the result is in range and free of `taken`.
func assertAllocated(t *testing.T, p *config.PortBlock, site string, taken map[int]bool) {
	t.Helper()
	port, err := AllocatePort(p, site, taken)
	if err != nil {
		t.Fatalf("AllocatePort: %v", err)
	}
	if port < p.Range[0] || port > p.Range[1] {
		t.Errorf("port %d outside range %v", port, p.Range)
	}
	if taken[port] {
		t.Errorf("port %d is in the taken set", port)
	}
}
