package tracer

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"golang.org/x/net/icmp"
	"net"
	"sync"
)

type ipTracer struct {
	conn *icmp.PacketConn
	mu   sync.Mutex

	preferIPv6 bool
}

func NewIPTracer(preferIPv6 bool) Tracer {
	return &ipTracer{preferIPv6: preferIPv6}
}

func (i *ipTracer) ListenConn(options *performOptions) (conn net.PacketConn, err error) {
	i.mu.Lock()
	defer func() {
		if i.conn == nil && err == nil {
			i.conn = conn.(*icmp.PacketConn)
		}
		i.mu.Unlock()
	}()
	if i.conn != nil {
		return i.conn, nil
	}
	switch options.network {
	case "ip":
		var proto string
		if i.preferIPv6 {
			proto = "ip6:ipv6-icmp"
		} else {
			proto = "ip4:icmp"
		}
		return icmp.ListenPacket(proto, options.iface)
	default:
		return nil, fmt.Errorf("unsupported network: %s", options.network)
	}
}

func (i *ipTracer) SendConn(options *performOptions) (conn net.PacketConn, err error) {
	return i.ListenConn(options)
}

func (i *ipTracer) SetTTLAndTOS(ttl, tos int) {
	i.mu.Lock()
	if conn4 := i.conn.IPv4PacketConn(); conn4 != nil {
		conn4.SetTTL(ttl)
		conn4.SetTOS(tos)
	}
	if conn6 := i.conn.IPv6PacketConn(); conn6 != nil {
		conn6.SetHopLimit(ttl)
		conn6.SetTrafficClass(tos)
	}
	i.mu.Unlock()
}

func (i *ipTracer) EchoPacket(id, seq uint16, packetSize int) ([]byte, error) {
	var serializableLayers []gopacket.SerializableLayer
	if i.preferIPv6 {
		serializableLayers = append(serializableLayers, &layers.ICMPv6{
			TypeCode: layers.CreateICMPv6TypeCode(layers.ICMPv6TypeEchoRequest, 0),
		})
	} else {
		serializableLayers = append(serializableLayers, &layers.ICMPv4{
			TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoRequest, 0),
			Id:       id,
			Seq:      seq,
		})
	}
	serializableLayers = append(serializableLayers, gopacket.Payload(byteSliceOfSize([]byte("HELLO-R-U-THERE"), packetSize)))

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	if err := gopacket.SerializeLayers(buffer, opts, serializableLayers...); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (i *ipTracer) CheckEchoReply(byte []byte, id, seq uint16) bool {
	tpe := layers.LayerTypeICMPv4
	if i.preferIPv6 {
		tpe = layers.LayerTypeICMPv6
	}
	packet := gopacket.NewPacket(byte, tpe, gopacket.Default)
	// TODO layers.ICMPv6 support
	if layer, ok := packet.Layer(tpe).(*layers.ICMPv4); ok {
		mt := layer.TypeCode.Type()
		if mt == layers.ICMPv4TypeEchoReply {
			return layer.Id == id && layer.Seq == seq
		} else if mt == layers.ICMPv4TypeTimeExceeded {
			packet := gopacket.NewPacket(layer.Payload, layers.IPProtocolIPv4, gopacket.Default)
			if layer, ok := packet.Layer(tpe).(*layers.ICMPv4); ok {
				return layer.Id == id && layer.Seq == seq
			}
		}
	}
	return false
}
