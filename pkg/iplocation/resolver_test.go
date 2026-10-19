package iplocation

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func databases() (string, string) {
	root := filepath.Join("..", "..", "data", "ipregion")
	return filepath.Join(root, "ip2region_v4.xdb"), filepath.Join(root, "ip2region_v6.xdb")
}

func testResolver(t *testing.T) *Resolver {
	t.Helper()
	v4, v6 := databases()
	r, err := New(v4, v6)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestBundledDatabasesMatchPinnedRelease(t *testing.T) {
	v4, v6 := databases()
	for path, expected := range map[string]string{
		v4: "8e31bbdccb5bf21028af10592d4312ec975da0bffa108c0c5d862a12190f9ad3",
		v6: "939f6b46bd2b8bec3cf7c5ceb8ba782266ae9b1f35b5ba7916700dec0b7506ed",
	} {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(content)
		if hex.EncodeToString(digest[:]) != expected {
			t.Fatalf("database does not match the documented release: %s", path)
		}
	}
}

func TestLookupBothIPVersions(t *testing.T) {
	r := testResolver(t)
	for _, tc := range []struct{ ip, contains string }{
		{"1.2.3.4", "Australia"},
		{"113.92.157.29", "广东"},
		{"240e:3b7:3272:d8d0:db09:c067:8d59:539e", "广东"},
		{"2604:a840:3::a04d", "United States"},
		{"::ffff:1.2.3.4", "Australia"},
	} {
		if got := r.Lookup(tc.ip); !strings.Contains(got, tc.contains) {
			t.Errorf("Lookup(%q) = %q, want region containing %q", tc.ip, got, tc.contains)
		}
	}
}

func TestLookupLocalAndInvalidAddresses(t *testing.T) {
	r := testResolver(t)
	for _, ip := range []string{"127.0.0.1", "::1", "10.2.3.4", "172.16.1.2", "192.168.1.1", "fc00::1", "fe80::1", "169.254.1.1", "100.64.0.1", "::ffff:192.168.1.1"} {
		if got := r.Lookup(ip); got != "内网" {
			t.Errorf("Lookup(%q) = %q", ip, got)
		}
	}
	for _, ip := range []string{"", "not-an-ip", "1.2.3.4:443", "1.2.3.4, 5.6.7.8", "999.0.0.1", "0.0.0.0", "::", "224.0.0.1", "ff02::1"} {
		if got := r.Lookup(ip); got != "未知" {
			t.Errorf("Lookup(%q) = %q", ip, got)
		}
	}
	var disabled *Resolver
	if got := disabled.Lookup("1.2.3.4"); got != "未知" {
		t.Fatalf("nil resolver = %q", got)
	}
}

func TestLookupConcurrent(t *testing.T) {
	r := testResolver(t)
	addresses := []string{"1.2.3.4", "113.92.157.29", "240e:3b7:3272:d8d0:db09:c067:8d59:539e", "2604:a840:3::a04d"}
	want := make(map[string]string, len(addresses))
	for _, ip := range addresses {
		want[ip] = r.Lookup(ip)
	}
	var workers sync.WaitGroup
	for i := 0; i < 32; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for n := 0; n < 100; n++ {
				for _, ip := range addresses {
					if got := r.Lookup(ip); got != want[ip] {
						t.Errorf("concurrent lookup changed for %s: %q", ip, got)
						return
					}
				}
			}
		}()
	}
	workers.Wait()
}

func TestNewRejectsMissingWrongVersionAndTruncatedDatabase(t *testing.T) {
	v4, v6 := databases()
	if _, err := New(filepath.Join(t.TempDir(), "missing.xdb"), v6); err == nil {
		t.Fatal("missing database accepted")
	}
	if _, err := New(v6, v4); err == nil {
		t.Fatal("reversed IP versions accepted")
	}
	data, err := os.ReadFile(v4)
	if err != nil {
		t.Fatal(err)
	}
	truncated := filepath.Join(t.TempDir(), "truncated.xdb")
	if err := os.WriteFile(truncated, data[:600000], 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(truncated, v6); err == nil {
		t.Fatal("truncated database accepted")
	}
}

func TestRegionDisplayAndMissingRegion(t *testing.T) {
	for raw, want := range map[string]string{
		"中国|广东省|深圳市|电信|CN":   "中国 / 广东省 / 深圳市",
		"中国|上海市|上海市|电信|CN":   "中国 / 上海市",
		"Australia|0|0|0|AU": "Australia",
		"0|0|0|ISP|ZZ":       "未知",
		"":                   "未知",
	} {
		if got := displayRegion(raw); got != want {
			t.Errorf("displayRegion(%q)=%q, want %q", raw, got, want)
		}
	}
	// An empty vector entry represents a real xdb lookup miss.
	r := testResolver(t)
	r.v4.content = make([]byte, len(r.v4.content))
	if got := r.Lookup("1.2.3.4"); got != "未知" {
		t.Fatalf("missing range = %q", got)
	}
}
