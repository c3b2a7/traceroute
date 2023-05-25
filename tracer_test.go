package tracer

import (
	"fmt"
	"net"
	"testing"
)

func TestTraceroute(t *testing.T) {
	addr, _ := net.ResolveIPAddr("ip", "14.119.104.189")
	recvCh := make(chan *TracerouteHop, 0)
	go func() {
		for {
			hop, ok := <-recvCh
			if !ok {
				fmt.Println()
				return
			}
			printHop(hop)
		}
	}()

	_, err := Traceroute(addr, recvCh)
	if err != nil {
		t.Error(err)
	}
}

func printHop(hop *TracerouteHop) {
	if hop.Success {
		host, err := net.LookupAddr(hop.From.String())
		if err == nil {
			fmt.Printf("%-3d (%s) %s    %v\n", hop.TTL, host, hop.From, hop.ElapsedTime)
		} else {
			fmt.Printf("%-3d %s    %v\n", hop.TTL, hop.From, hop.ElapsedTime)
		}
	} else {
		fmt.Printf("%-3d *\n", hop.TTL)
	}
}
