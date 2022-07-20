package utils

import (
	"log"
	"net"
	"net/http"
	"net/netip"
)

func GetIP(r *http.Request) (ip string) {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Println("failed get remote addr, err:", err)
		return ""
	}

	xip := r.Header.Get("X-Real-IP")
	if IPisLocal(ip) && xip != "" {
		return xip
	}

	return ip
}

func CidrMatch(net string, ip string) bool {
	network, err := netip.ParsePrefix(net)
	if err != nil {
		log.Println(err)
		return false
	}

	ipp, err := netip.ParseAddr(ip)
	if err != nil {
		log.Println(err)
		return false
	}

	return network.Contains(ipp)
}

func IPisLocal(ip string) bool {
	if CidrMatch("127.0.0.0/8", ip) || CidrMatch("::1/128", ip) || // loop back
		CidrMatch("172.16.0.0/12", ip) || CidrMatch("10.0.0.0/8", ip) || // local
		CidrMatch("192.168.0.0/16", ip) || CidrMatch("fc00::/7", ip) {
		return true
	}
	return false
}
