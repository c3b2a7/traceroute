package tracer

import (
	"net"
	"os"
	"time"
)

func byteSliceOfSize(buf []byte, n int) []byte {
	padding := n - len(buf)
	if padding < 0 {
		return buf
	}
	b := make([]byte, padding)
	for i := 0; i < padding; i++ {
		b[i] = 1
	}
	return append(buf, b...)
}

func isIPv4(addr *net.IPAddr) bool {
	return len(addr.IP.To4()) == net.IPv4len
}

func isIPv6(addr *net.IPAddr) bool {
	return addr.IP.To4() == nil && len(addr.IP.To16()) == net.IPv6len
}

type Tracer interface {
	ListenConn(options *performOptions) (net.PacketConn, error)
	SendConn(options *performOptions) (net.PacketConn, error)
	SetTTLAndTOS(ttl, tos int)
	EchoPacket(id, seq uint16, packetSize int) ([]byte, error)
	CheckEchoReply(byte []byte, id, seq uint16) bool
}

type TracerouteHop struct {
	Success     bool
	From        net.Addr
	N, TTL      int
	ElapsedTime time.Duration
}

type TracerouteResult struct {
	hops     []*TracerouteHop
	destAddr net.Addr
}

func Traceroute(target *net.IPAddr, recvCh chan<- *TracerouteHop, option ...PerformOption) (result TracerouteResult, err error) {
	var options = &performOptions{
		network: "ip",
		iface:   "",

		firstTTL:   1,
		maxTTL:     64,
		tos:        0,
		packetSize: 54,
		queries:    3,

		timeout: 500 * time.Millisecond,
	}
	for _, opt := range option {
		opt(options)
	}

	result.destAddr = target

	var id, seq uint16 = uint16(os.Getpid() & 0xffff), 1
	ttl, retry := options.firstTTL, 0

	for {
		start := time.Now()

		// TODO support other protocol conn
		tracer := NewIPTracer(isIPv6(target))

		var sendConn net.PacketConn
		var recvConn net.PacketConn
		if sendConn, err = tracer.SendConn(options); err != nil {
			return
		}
		if recvConn, err = tracer.ListenConn(options); err != nil {
			return
		}
		defer recvConn.Close()
		defer sendConn.Close()

		// set ttl and tos for future outgoing packets
		tracer.SetTTLAndTOS(ttl, options.tos)
		// generate echo request packet
		var packet []byte

		if packet, err = tracer.EchoPacket(id, seq, options.packetSize); err != nil {
			return
		}
		// write packet
		sendConn.WriteTo(packet, target)

		// receive reply
		recvConn.SetReadDeadline(time.Now().Add(options.timeout))
		var recvBuf = make([]byte, options.packetSize)
		n, from, err := recvConn.ReadFrom(recvBuf)
		elapsed := time.Since(start)

		if err == nil && tracer.CheckEchoReply(recvBuf[:n], id, seq) {
			hop := &TracerouteHop{
				Success:     true,
				From:        from,
				N:           n,
				TTL:         ttl,
				ElapsedTime: elapsed,
			}
			recvCh <- hop
			result.hops = append(result.hops, hop)

			ttl++
			retry = 0
		} else {
			retry++
			if retry > options.queries {
				recvCh <- &TracerouteHop{Success: false, TTL: ttl}
				ttl++
				retry = 0
			}
		}

		if ttl > options.maxTTL || (from != nil && target.IP.String() == from.String()) {
			close(recvCh)
			return result, nil
		}

		seq++
	}
}
