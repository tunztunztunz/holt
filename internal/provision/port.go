package provision

import (
	"fmt"
	"hash/fnv"
	"net"

	"github.com/tunztunztunz/acre/internal/config"
)

func AllocatePort(p *config.PortBlock, siteName string, taken map[int]bool) (int, error) {
	if p == nil {
		return 0, nil // Port is optional
	}
	lo, hi := p.Range[0], p.Range[1]
	span := hi - lo + 1

	switch p.Strategy {
	case "", config.PortHash:
		h := fnv.New32a()
		_, err := h.Write([]byte(siteName))
		if err != nil {
			return 0, err
		}

		start := lo + int(h.Sum32())%span

		for i := range span {
			port := lo + (start-lo+i)%span
			if taken[port] || !isFree(port) {
				continue
			}
			return port, nil
		}

		return 0, fmt.Errorf("no free port available in range %d-%d", lo, hi)

	case config.PortFree:
		for port := lo; port <= hi; port++ {
			if taken[port] || !isFree(port) {
				continue
			}
			return port, nil
		}
		return 0, fmt.Errorf("no free port available in range %d-%d", lo, hi)

	default:
		return 0, fmt.Errorf("unknown port strategy: %s", p.Strategy)

	}
}

func isFree(port int) bool {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	_ = l.Close()
	return true
}
