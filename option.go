package tracer

import (
	"net/netip"
	"time"
)

type performOptions struct {
	network string
	iface   string

	firstTTL   int
	maxTTL     int
	tos        int
	packetSize int
	queries    int

	timeout time.Duration
}

type PerformOption func(options *performOptions)

func WithNetwork(network string) PerformOption {
	return func(options *performOptions) {
		switch network {
		case "ip":
			fallthrough
		case "udp":
			options.network = network
		}
	}
}

func WithInterface(iface string) PerformOption {
	return func(options *performOptions) {
		if _, err := netip.ParseAddr(iface); err == nil {
			options.iface = iface
		}
	}
}

func WithFirstTTL(firstTTL int) PerformOption {
	return func(options *performOptions) {
		options.firstTTL = firstTTL
	}
}

func WithMaxTTL(maxTTL int) PerformOption {
	return func(options *performOptions) {
		options.maxTTL = maxTTL
	}
}

func WithPacketSize(packetSize int) PerformOption {
	return func(options *performOptions) {
		options.packetSize = packetSize
	}
}

func WithQueries(queries int) PerformOption {
	return func(options *performOptions) {
		options.queries = queries
	}
}

func WithTOS(tos int) PerformOption {
	return func(options *performOptions) {
		options.tos = tos
	}
}

func WithTimeout(timeout time.Duration) PerformOption {
	return func(options *performOptions) {
		options.timeout = timeout
	}
}
