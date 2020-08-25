/*
 * NETCAP - Traffic Analysis Framework
 * Copyright (c) 2017-2020 Philipp Mieden <dreadl0ck [at] protonmail [dot] ch>
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package decoder

import (
	"log"
	"sync"
	"sync/atomic"

	"github.com/dreadl0ck/gopacket"
	"github.com/dreadl0ck/gopacket/layers"
	"github.com/dreadl0ck/ja3"
	"github.com/dreadl0ck/netcap/dpi"
	"github.com/dreadl0ck/netcap/resolvers"
	"github.com/dreadl0ck/netcap/types"
	"github.com/dreadl0ck/tlsx"
	"github.com/gogo/protobuf/proto"
)

var (
	// LocalDNS controls whether the DNS names shall be resolved locally
	// without contacting a nameserver.
	LocalDNS = true

	ipProfileDecoderInstance *customDecoder
	ipProfiles               int64
)

// atomicIPProfileMap contains all connections and provides synchronized access.
type atomicIPProfileMap struct {
	// SrcIP to DeviceProfiles
	Items map[string]*ipProfile
	sync.Mutex
}

// Size returns the number of elements in the Items map.
func (a *atomicIPProfileMap) Size() int {
	a.Lock()
	defer a.Unlock()

	return len(a.Items)
}

// IPProfiles contains a map of IP specific behavior profiles at runtime.
var IPProfiles = &atomicIPProfileMap{
	Items: make(map[string]*ipProfile),
}

// wrapper for the types.IPProfile that can be locked.
type ipProfile struct {
	*types.IPProfile
	sync.Mutex
}

var ipProfileDecoder = newCustomDecoder(
	types.Type_NC_IPProfile,
	"IPProfile",
	"An IPProfile contains information about a single IPv4 or IPv6 address seen on the network and it's behavior",
	func(d *customDecoder) error {
		ipProfileDecoderInstance = d

		return nil
	},
	func(p gopacket.Packet) proto.Message {
		return nil
	},
	func(e *customDecoder) error {
		// teardown DPI C libs
		dpi.Destroy()

		// flush writer
		for _, item := range IPProfiles.Items {
			item.Lock()
			writeIPProfile(item.IPProfile)
			item.Unlock()
		}

		return nil
	},
)

// GetIPProfile fetches a known profile and updates it or returns a new one.
func getIPProfile(ipAddr string, i *packetInfo) *ipProfile {
	if ipAddr == "" {
		return nil
	}

	IPProfiles.Lock()
	if p, ok := IPProfiles.Items[ipAddr]; ok {
		IPProfiles.Unlock()

		p.Lock()

		p.NumPackets++
		p.TimestampLast = i.timestamp

		dataLen := uint64(len(i.p.Data()))
		p.Bytes += dataLen

		// Transport Layer
		if tl := i.p.TransportLayer(); tl != nil {
			var port *types.Port

			if port, ok = p.SrcPorts[tl.TransportFlow().Src().String()]; ok {
				atomic.AddUint64(&port.NumTotal, dataLen)

				if tl.LayerType() == layers.LayerTypeTCP {
					atomic.AddUint64(&port.NumTCP, 1)
				} else if tl.LayerType() == layers.LayerTypeUDP {
					atomic.AddUint64(&port.NumUDP, 1)
				}
			} else {
				port = &types.Port{
					NumTotal: dataLen,
				}
				if tl.LayerType() == layers.LayerTypeTCP {
					port.NumTCP++
				} else if tl.LayerType() == layers.LayerTypeUDP {
					port.NumUDP++
				}
				p.SrcPorts[tl.TransportFlow().Src().String()] = port
			}

			if port, ok = p.DstPorts[tl.TransportFlow().Dst().String()]; ok {
				port.NumTotal += dataLen
				if tl.LayerType() == layers.LayerTypeTCP {
					port.NumTCP++
				} else if tl.LayerType() == layers.LayerTypeUDP {
					port.NumUDP++
				}
			} else {
				port = &types.Port{
					NumTotal: dataLen,
				}
				if tl.LayerType() == layers.LayerTypeTCP {
					port.NumTCP++
				} else if tl.LayerType() == layers.LayerTypeUDP {
					port.NumUDP++
				}
				p.DstPorts[tl.TransportFlow().Dst().String()] = port
			}
		}

		// Session Layer: TLS
		ch := tlsx.GetClientHelloBasic(i.p)
		if ch != nil {
			if ch.SNI != "" {
				p.SNIs[ch.SNI]++
			}
		}

		ja3Hash := ja3.DigestHexPacket(i.p)
		if ja3Hash == "" {
			ja3Hash = ja3.DigestHexPacketJa3s(i.p)
		}

		if ja3Hash != "" {
			// add hash to profile if not already present
			if _, ok = p.Ja3[ja3Hash]; !ok {
				p.Ja3[ja3Hash] = resolvers.LookupJa3(ja3Hash)
			}
		}

		// Application Layer: DPI
		uniqueResults := dpi.GetProtocols(i.p)
		for proto, res := range uniqueResults {
			// check if proto exists already
			var prot *types.Protocol
			if prot, ok = p.Protocols[proto]; ok {
				prot.Packets++
			} else {
				// add new
				p.Protocols[proto] = dpi.NewProto(&res)
			}
		}

		p.Unlock()

		return p
	}
	IPProfiles.Unlock()

	var (
		protos   = make(map[string]*types.Protocol)
		ja3Map   = make(map[string]string)
		dataLen  = uint64(len(i.p.Data()))
		srcPorts = make(map[string]*types.Port)
		dstPorts = make(map[string]*types.Port)
		sniMap   = make(map[string]int64)
	)

	// Network Layer: IP Geolocation
	loc, _ := resolvers.LookupGeolocation(ipAddr)

	// Transport Layer: Port information

	if tl := i.p.TransportLayer(); tl != nil {
		srcPort := &types.Port{
			NumTotal: dataLen,
		}

		if tl.LayerType() == layers.LayerTypeTCP {
			srcPort.NumTCP++
		} else if tl.LayerType() == layers.LayerTypeUDP {
			srcPort.NumUDP++
		}

		srcPorts[tl.TransportFlow().Src().String()] = srcPort

		dstPort := &types.Port{
			NumTotal: dataLen,
		}
		if tl.LayerType() == layers.LayerTypeTCP {
			dstPort.NumTCP++
		} else if tl.LayerType() == layers.LayerTypeUDP {
			dstPort.NumUDP++
		}
		dstPorts[tl.TransportFlow().Dst().String()] = dstPort
	}

	// Session Layer: TLS

	ja3Hash := ja3.DigestHexPacket(i.p)
	if ja3Hash == "" {
		ja3Hash = ja3.DigestHexPacketJa3s(i.p)
	}

	if ja3Hash != "" {
		ja3Map[ja3Hash] = resolvers.LookupJa3(ja3Hash)
	}

	ch := tlsx.GetClientHelloBasic(i.p)
	if ch != nil {
		sniMap[ch.SNI] = 1
	}

	// Application Layer: DPI
	uniqueResults := dpi.GetProtocols(i.p)
	for proto, res := range uniqueResults {
		protos[proto] = dpi.NewProto(&res)
	}

	var names []string
	if LocalDNS {
		if name := resolvers.LookupDNSNameLocal(ipAddr); len(name) != 0 {
			names = append(names, name)
		}
	} else {
		names = resolvers.LookupDNSNames(ipAddr)
	}

	// create new profile
	p := &ipProfile{
		IPProfile: &types.IPProfile{
			Addr:           ipAddr,
			NumPackets:     1,
			Geolocation:    loc,
			DNSNames:       names,
			TimestampFirst: i.timestamp,
			Ja3:            ja3Map,
			Protocols:      protos,
			Bytes:          dataLen,
			SrcPorts:       srcPorts,
			DstPorts:       dstPorts,
			SNIs:           sniMap,
		},
	}

	IPProfiles.Lock()
	IPProfiles.Items[ipAddr] = p
	IPProfiles.Unlock()

	return p
}

// writeIPProfile writes the ip profile.
func writeIPProfile(i *types.IPProfile) {
	if conf.ExportMetrics {
		i.Inc()
	}

	atomic.AddInt64(&ipProfileDecoderInstance.numRecords, 1)

	err := ipProfileDecoderInstance.writer.Write(i)
	if err != nil {
		log.Fatal("failed to write proto: ", err)
	}
}
