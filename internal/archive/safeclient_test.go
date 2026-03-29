package archive

import (
	"net"
	"testing"
)

func TestIsPublicIP(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		// Should block
		{"127.0.0.1", false},
		{"127.0.0.2", false},
		{"10.0.0.1", false},
		{"10.255.255.255", false},
		{"172.16.0.1", false},
		{"172.31.255.255", false},
		{"192.168.0.1", false},
		{"192.168.1.1", false},
		{"169.254.169.254", false}, // AWS metadata
		{"169.254.0.1", false},     // link-local
		{"0.0.0.0", false},
		{"::1", false},             // IPv6 loopback
		{"fe80::1", false},         // IPv6 link-local
		{"fc00::1", false},         // IPv6 unique local
		{"fd00::1", false},         // IPv6 unique local

		// Should allow
		{"8.8.8.8", true},
		{"1.1.1.1", true},
		{"104.16.0.1", true},
		{"185.199.108.153", true},
		{"2606:4700::1", true}, // Cloudflare IPv6
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		if ip == nil {
			t.Fatalf("failed to parse IP: %s", tt.ip)
		}
		got := isPublicIP(ip)
		if got != tt.want {
			t.Errorf("isPublicIP(%s) = %v, want %v", tt.ip, got, tt.want)
		}
	}
}

func TestSafeClientBlocksLocalhost(t *testing.T) {
	client := safeClient()
	_, err := client.Get("http://127.0.0.1:9999/test")
	if err == nil {
		t.Fatal("expected error when connecting to localhost")
	}
}

func TestSafeClientBlocksMetadata(t *testing.T) {
	client := safeClient()
	_, err := client.Get("http://169.254.169.254/latest/meta-data/")
	if err == nil {
		t.Fatal("expected error when connecting to metadata endpoint")
	}
}

func TestSafeClientBlocksPrivateIP(t *testing.T) {
	client := safeClient()
	_, err := client.Get("http://10.0.0.1:8080/internal")
	if err == nil {
		t.Fatal("expected error when connecting to private IP")
	}
}
