package types

import "github.com/prometheus/client_golang/prometheus"

// Metrics contains all available prometheus collectors.
var Metrics = []prometheus.Collector{
	sipMetric,
	enipMetric,
	credentialsMetric,
	llcMetric,
	ipSecEspMetric,
	tlsClientMetric,
	dnsMetric,
	ethernetCTPMetric,
	ethernetMetric,
	ethernetPayloadEntropy,
	ethernetPayloadSize,
	dhcp4Metric,
	serviceMetric,
	icmp6raMetric,
	eapMetric,
	ipv6fragMetric,
	icmp6Metric,
	tlsServerMetric,
	ntpMetric,
	sctpMetric,
	flowMetric,
	flowTotalSize,
	flowAppPayloadSize,
	flowNumPackets,
	flowDuration,
	ciscoDiscoveryMetric,
	usbRequestBlockSetupMetric,
	mplsMetric,
	icmp6rsMetric,
	snapMetric,
	eapPolKeyMetric,
	geneveMetric,
	ipSecAhMetric,
	ip4Metric,
	ip4PayloadEntropy,
	ip4PayloadSize,
	vrrp2Metric,
	ethernetCTPReplyMetric,
	igmpMetric,
	greMetric,
	ip6hopMetric,
	vxlanMetric,
	modbusTCPMetric,
	smtpMetric,
	ciscoDiscoveryInfoMetric,
	arpMetric,
	httpMetric,
	lldiMetric,
	ip6Metric,
	ip6PayloadEntropy,
	ip6PayloadSize,
	fileMetric,
	udpMetric,
	udpPayloadEntropy,
	udpPayloadSize,
	cipMetric,
	lcmMetric,
	pop3Metric,
	connectionsMetric,
	connTotalSize,
	connAppPayloadSize,
	connNumPackets,
	connDuration,
	dot11Metric,
	tcpMetric,
	tcpPayloadEntropy,
	tcpPayloadSize,
	icmp6nsMetric,
	softwareMetric,
	fddiMetric,
	eapPolMetric,
	diameterMetric,
	dot1qMetric,
	ospf3Metric,
	exploitMetric,
	nortelDiscoveryMetric,
	vulnerabilityMetric,
	usbMetric,
	ospf2Metric,
	icmp4Metric,
	sshMetric,
	icmp6eMetric,
	icmp6naMetric,
	lldMetric,
	dhcp6Metric,
	bfdMetric,
	ethernetMetric,
	ethernetPayloadEntropy,
	ethernetPayloadSize,
	flowMetric,
	flowTotalSize,
	flowAppPayloadSize,
	flowNumPackets,
	flowDuration,
	ip4Metric,
	ip4PayloadEntropy,
	ip4PayloadSize,
	ip6Metric,
	ip6PayloadEntropy,
	ip6PayloadSize,
	udpMetric,
	udpPayloadEntropy,
	udpPayloadSize,
	connectionsMetric,
	connTotalSize,
	connAppPayloadSize,
	connNumPackets,
	connDuration,
	tcpMetric,
	tcpPayloadEntropy,
	tcpPayloadSize,
}
