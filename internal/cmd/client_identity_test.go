package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func captureRequestLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &logs
}

func TestClientIdentityTrustAndValidation(t *testing.T) {
	for _, tc := range []struct {
		name     string
		peer     string
		ips      []string
		ports    []string
		wantIP   string
		wantPort float64
	}{
		{"ipv4", "10.0.4.12:12345", []string{"203.0.113.9"}, []string{"54321"}, "203.0.113.9", 54321},
		{"ipv6", "[fd00::12]:12345", []string{"2001:db8::9"}, []string{"65535"}, "2001:db8::9", 65535},
		{"mapped ipv4 peer", "[::ffff:10.0.4.12]:12345", []string{"203.0.113.9"}, nil, "203.0.113.9", 0},
		{"mapped ipv4 client", "10.0.4.12:12345", []string{"::ffff:203.0.113.9"}, nil, "203.0.113.9", 0},
		{"mapped unspecified client", "10.0.4.12:12345", []string{"::ffff:0.0.0.0"}, nil, "", 0},
		{"mapped multicast client", "10.0.4.12:12345", []string{"::ffff:224.0.0.1"}, nil, "", 0},
		{"untrusted peer", "198.51.100.1:12345", []string{"203.0.113.9"}, []string{"54321"}, "", 0},
		{"outside trusted subnet", "10.0.8.12:12345", []string{"203.0.113.9"}, nil, "", 0},
		{"malformed peer", "not-an-address", []string{"203.0.113.9"}, nil, "", 0},
		{"missing ip", "10.0.4.12:12345", nil, []string{"54321"}, "", 0},
		{"empty ip", "10.0.4.12:12345", []string{""}, nil, "", 0},
		{"invalid ip", "10.0.4.12:12345", []string{"not-an-ip"}, nil, "", 0},
		{"ip with port", "10.0.4.12:12345", []string{"203.0.113.9:54321"}, nil, "", 0},
		{"ip list", "10.0.4.12:12345", []string{"203.0.113.9, 198.51.100.1"}, nil, "", 0},
		{"duplicate ip", "10.0.4.12:12345", []string{"203.0.113.9", "203.0.113.9"}, nil, "", 0},
		{"scoped ip", "10.0.4.12:12345", []string{"fe80::1%eth0"}, nil, "", 0},
		{"unspecified ip", "10.0.4.12:12345", []string{"0.0.0.0"}, nil, "", 0},
		{"multicast ip", "10.0.4.12:12345", []string{"ff02::1"}, nil, "", 0},
		{"missing port", "10.0.4.12:12345", []string{"203.0.113.9"}, nil, "203.0.113.9", 0},
		{"empty port", "10.0.4.12:12345", []string{"203.0.113.9"}, []string{""}, "203.0.113.9", 0},
		{"invalid port", "10.0.4.12:12345", []string{"203.0.113.9"}, []string{"https"}, "203.0.113.9", 0},
		{"zero port", "10.0.4.12:12345", []string{"203.0.113.9"}, []string{"0"}, "203.0.113.9", 0},
		{"large port", "10.0.4.12:12345", []string{"203.0.113.9"}, []string{"65536"}, "203.0.113.9", 0},
		{"negative port", "10.0.4.12:12345", []string{"203.0.113.9"}, []string{"-1"}, "203.0.113.9", 0},
		{"duplicate port", "10.0.4.12:12345", []string{"203.0.113.9"}, []string{"1", "2"}, "203.0.113.9", 0},
		{"port list", "10.0.4.12:12345", []string{"203.0.113.9"}, []string{"1, 2"}, "203.0.113.9", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logs := captureRequestLogs(t)
			handler, err := withClientIdentity([]string{"10.0.4.0/22", "fd00::/64"}, http.HandlerFunc(handleGetPing))
			if err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest(http.MethodGet, "/ping", nil)
			r.RemoteAddr = tc.peer
			for _, ip := range tc.ips {
				r.Header.Add("X-Echo-Client-IP", ip)
			}
			for _, port := range tc.ports {
				r.Header.Add("X-Echo-Client-Port", port)
			}
			// None of these may override the dedicated identity contract or fill a gap.
			r.Header.Set("X-Forwarded-For", "192.0.2.99")
			r.Header.Set("X-Real-IP", "192.0.2.99")
			r.Header.Set("Forwarded", "for=192.0.2.99")
			r.Header.Set("X-Forwarded-Port", "443")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != http.StatusOK || w.Body.String() != "pong" {
				t.Fatalf("identity affected response: %d %q", w.Code, w.Body.String())
			}
			var entry map[string]any
			if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
				t.Fatal(err)
			}
			if entry["remoteAddr"] != tc.peer {
				t.Errorf("remoteAddr = %v, want %s", entry["remoteAddr"], tc.peer)
			}
			if tc.wantIP == "" {
				if _, ok := entry["clientIp"]; ok {
					t.Errorf("unexpected clientIp: %v", entry["clientIp"])
				}
			} else if entry["clientIp"] != tc.wantIP {
				t.Errorf("clientIp = %v, want %s", entry["clientIp"], tc.wantIP)
			}
			if tc.wantPort == 0 {
				if _, ok := entry["clientPort"]; ok {
					t.Errorf("unexpected clientPort: %v", entry["clientPort"])
				}
			} else if entry["clientPort"] != tc.wantPort {
				t.Errorf("clientPort = %v, want %v", entry["clientPort"], tc.wantPort)
			}
		})
	}
}

func TestNoTrustedProxiesByDefault(t *testing.T) {
	logs := captureRequestLogs(t)
	handler, err := withClientIdentity(nil, http.HandlerFunc(handleGetPing))
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, "/ping", nil)
	r.RemoteAddr = "10.0.4.12:12345"
	r.Header.Set("X-Echo-Client-IP", "203.0.113.9")
	handler.ServeHTTP(httptest.NewRecorder(), r)
	var entry map[string]any
	if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}
	if _, ok := entry["clientIp"]; ok {
		t.Fatal("forwarded identity trusted without configured CIDRs")
	}
}

func TestInvalidTrustedProxyConfiguration(t *testing.T) {
	for _, cidr := range []string{"", "invalid", "10.0.4.1", "0.0.0.0/0", "::/0", "::ffff:10.0.4.0/120"} {
		if _, err := withClientIdentity([]string{cidr}, http.HandlerFunc(handleGetPing)); err == nil {
			t.Errorf("accepted invalid CIDR %q", cidr)
		}
	}
}

func TestEveryRequestLogIncludesIdentity(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, body, response string
		handler                            http.HandlerFunc
	}{
		{"get", "GET", "/?message=hello", "", "Received message (GET): hello", handleGetEcho},
		{"get missing message", "GET", "/", "", "Missing 'message' parameter.", handleGetEcho},
		{"post", "POST", "/", `{"message":"hello"}`, "Received message (POST): hello", handlePostEcho},
		{"post missing message", "POST", "/", `{}`, "Missing 'message' parameter.", handlePostEcho},
		{"post malformed json", "POST", "/", `invalid`, "body is formatted incorrectly", handlePostEcho},
		{"ping", "GET", "/ping", "", "pong", handleGetPing},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logs := captureRequestLogs(t)
			handler, err := withClientIdentity([]string{"10.0.4.0/22"}, tc.handler)
			if err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			r.RemoteAddr = "10.0.4.12:12345"
			r.Header.Set("X-Echo-Client-IP", "203.0.113.9")
			r.Header.Set("X-Echo-Client-Port", "54321")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != http.StatusOK || w.Body.String() != tc.response {
				t.Fatalf("changed response: %d %q", w.Code, w.Body.String())
			}
			var entry map[string]any
			if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
				t.Fatal(err)
			}
			if entry["clientIp"] != "203.0.113.9" || entry["clientPort"] != float64(54321) || entry["remoteAddr"] != r.RemoteAddr {
				t.Errorf("missing address fields: %v", entry)
			}
		})
	}
}
