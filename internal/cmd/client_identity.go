package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
)

type requestLoggerKey struct{}

// withClientIdentity trusts only the immediate connection peer. The ingress
// must overwrite both dedicated headers, even when PROXY metadata is absent.
// Trusted CIDRs include every caller in those networks, not just ingress pods.
func withClientIdentity(cidrs []string, next http.Handler) (http.Handler, error) {
	trusted := make([]netip.Prefix, 0, len(cidrs))
	for _, cidr := range cidrs {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(cidr))
		if err != nil || prefix.Bits() == 0 || prefix.Addr().Is4In6() {
			return nil, fmt.Errorf("invalid trusted proxy CIDR %q: use a nonzero IPv4 or IPv6 prefix", cidr)
		}
		trusted = append(trusted, prefix.Masked())
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attrs := clientIdentityAttrs(r, trusted)
		if len(attrs) > 0 {
			logger := slog.Default().With(attrs...)
			r = r.WithContext(context.WithValue(r.Context(), requestLoggerKey{}, logger))
		}
		next.ServeHTTP(w, r)
	}), nil
}

func requestLogger(r *http.Request) *slog.Logger {
	if logger, ok := r.Context().Value(requestLoggerKey{}).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

func clientIdentityAttrs(r *http.Request, trusted []netip.Prefix) []any {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return nil
	}
	peer, err := netip.ParseAddr(host)
	if err != nil || peer.Zone() != "" {
		return nil
	}
	allowed := false
	for _, prefix := range trusted {
		if prefix.Contains(peer.Unmap()) {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil
	}

	// Accept exactly one address, never a caller-controlled forwarding chain.
	ips := r.Header.Values("X-Echo-Client-IP")
	if len(ips) != 1 {
		return nil
	}
	ip, err := netip.ParseAddr(ips[0])
	if err != nil || ip.Zone() != "" {
		return nil
	}
	ip = ip.Unmap()
	if ip.IsUnspecified() || ip.IsMulticast() {
		return nil
	}
	attrs := []any{slog.String("clientIp", ip.String())}
	ports := r.Header.Values("X-Echo-Client-Port")
	if len(ports) == 1 {
		port, err := strconv.ParseUint(ports[0], 10, 16)
		if err == nil && port > 0 {
			attrs = append(attrs, slog.Int("clientPort", int(port)))
		}
	}
	return attrs
}
