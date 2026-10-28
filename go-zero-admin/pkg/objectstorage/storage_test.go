package objectstorage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func storageConfig(endpoint string) Config {
	return Config{Endpoint: endpoint, ExternalEndpoint: "http://127.0.0.1:9000", Region: "us-east-1", AccessKeyID: "test-access", SecretAccessKey: "test-secret", PublicBucket: "public-images", PrivateBucket: "private-images"}
}

// A virtual-hosted request, an ACL, a wrong bucket or a request sent to the
// external endpoint must fail this test rather than silently changing access.
func TestRustFSWritesAndDeletesUseInternalPathStyleAndVisibilityBuckets(t *testing.T) {
	type request struct{ method, path, host, mime, acl, body string }
	requests := make(chan request, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requests <- request{r.Method, r.URL.Path, r.Host, r.Header.Get("Content-Type"), r.Header.Get("X-Amz-Acl"), string(body)}
		if !strings.Contains(r.Header.Get("Authorization"), "Credential=test-access/") {
			t.Error("request was not signed with configured credentials")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	client, err := New(storageConfig(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	for _, visibility := range []string{"public", "private"} {
		if err := client.Put(context.Background(), "go-zero-admin/image.png", "image/png", []byte("image bytes"), visibility); err != nil {
			t.Fatal(err)
		}
		if err := client.Delete(context.Background(), "go-zero-admin/image.png", visibility); err != nil {
			t.Fatal(err)
		}
	}
	wantPaths := []string{"/public-images/go-zero-admin/image.png", "/public-images/go-zero-admin/image.png", "/private-images/go-zero-admin/image.png", "/private-images/go-zero-admin/image.png"}
	for i, want := range wantPaths {
		got := <-requests
		if got.path != want || got.host != strings.TrimPrefix(server.URL, "http://") || got.acl != "" {
			t.Fatalf("unexpected S3 request: %+v", got)
		}
		if i%2 == 0 && (got.method != "PUT" || got.mime != "image/png" || got.body != "image bytes") {
			t.Fatalf("upload body or MIME lost: %+v", got)
		}
		if i%2 == 1 && got.method != "DELETE" {
			t.Fatalf("expected DELETE: %+v", got)
		}
	}
}

func TestRustFSURLsUseExternalEndpointAndPrivateSignatureLifetime(t *testing.T) {
	config := storageConfig("http://rustfs:9000")
	client, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	publicURL, expires, err := client.URL(context.Background(), "go-zero-admin/a b.png", "public")
	if err != nil || publicURL != "http://127.0.0.1:9000/public-images/go-zero-admin/a%20b.png" || expires != 0 {
		t.Fatalf("public URL=%q expires=%d err=%v", publicURL, expires, err)
	}
	link, expires, err := client.URL(context.Background(), "go-zero-admin/a b.png", "private")
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	if u.Scheme != "http" || u.Host != "127.0.0.1:9000" || u.Path != "/private-images/go-zero-admin/a b.png" || expires != 300 {
		t.Fatalf("wrong private URL host/path/expiry: %s %d", link, expires)
	}
	if u.Query().Get("X-Amz-Expires") != "300" || u.Query().Get("X-Amz-Signature") == "" || !strings.HasPrefix(u.Query().Get("X-Amz-Credential"), "test-access/") {
		t.Fatal("missing signed credentials or default expiry")
	}
	if strings.Contains(link, config.SecretAccessKey) || strings.Contains(link, "rustfs") {
		t.Fatal("private URL leaked secret or internal endpoint")
	}
	assertPrivateSignature(t, u, config.SecretAccessKey)
	config.PresignTTLSeconds = 90
	client, err = New(config)
	if err != nil {
		t.Fatal(err)
	}
	link, expires, err = client.URL(context.Background(), "go-zero-admin/image.png", "private")
	if err != nil || expires != 90 || !strings.Contains(link, "X-Amz-Expires=90") {
		t.Fatalf("configured lifetime ignored: %s %d %v", link, expires, err)
	}
}

// Independently verify the SigV4 signature over the browser-facing host. A
// signed internal URL with only its hostname replaced must fail verification.
func assertPrivateSignature(t *testing.T, u *url.URL, secret string) {
	t.Helper()
	query := u.Query()
	signature := query.Get("X-Amz-Signature")
	query.Del("X-Amz-Signature")
	credential := strings.Split(query.Get("X-Amz-Credential"), "/")
	if len(credential) != 5 || query.Get("X-Amz-SignedHeaders") != "host" {
		t.Fatal("unexpected signing scope or headers")
	}
	canonical := "GET\n" + u.EscapedPath() + "\n" + strings.ReplaceAll(query.Encode(), "+", "%20") + "\nhost:" + u.Host + "\n\nhost\nUNSIGNED-PAYLOAD"
	digest := sha256.Sum256([]byte(canonical))
	toSign := "AWS4-HMAC-SHA256\n" + query.Get("X-Amz-Date") + "\n" + strings.Join(credential[1:], "/") + "\n" + hex.EncodeToString(digest[:])
	signingKey := []byte("AWS4" + secret)
	for _, value := range credential[1:] {
		mac := hmac.New(sha256.New, signingKey)
		_, _ = mac.Write([]byte(value))
		signingKey = mac.Sum(nil)
	}
	mac := hmac.New(sha256.New, signingKey)
	_, _ = mac.Write([]byte(toSign))
	if hex.EncodeToString(mac.Sum(nil)) != signature {
		t.Fatal("private URL signature does not authenticate the external endpoint")
	}
}

func TestRustFSRejectsPartialOrUnsafeConfigurationWithoutLeakingSecrets(t *testing.T) {
	client, err := New(Config{})
	if err != nil || client != nil {
		t.Fatal("absent storage must remain optional")
	}
	if _, _, err := client.URL(context.Background(), "key", "public"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("unconfigured URL did not fail clearly: %v", err)
	}
	if err := client.Put(context.Background(), "key", "image/png", []byte("data"), "public"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("unconfigured upload did not fail clearly: %v", err)
	}
	if err := client.Delete(context.Background(), "key", "public"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("unconfigured delete did not fail clearly: %v", err)
	}
	for name, mutate := range map[string]func(*Config){
		"missing internal endpoint": func(c *Config) { c.Endpoint = "" },
		"missing external endpoint": func(c *Config) { c.ExternalEndpoint = "" },
		"missing region":            func(c *Config) { c.Region = "" },
		"missing key":               func(c *Config) { c.AccessKeyID = "" },
		"missing secret":            func(c *Config) { c.SecretAccessKey = "" },
		"missing public bucket":     func(c *Config) { c.PublicBucket = "" },
		"missing private bucket":    func(c *Config) { c.PrivateBucket = "" },
		"same buckets":              func(c *Config) { c.PrivateBucket = c.PublicBucket },
		"invalid endpoint":          func(c *Config) { c.Endpoint = "ftp://rustfs:9000" },
		"endpoint credentials":      func(c *Config) { c.Endpoint = "http://test-access:test-secret@rustfs:9000" },
		"external endpoint query":   func(c *Config) { c.ExternalEndpoint += "?secret=test-secret" },
		"unsafe bucket":             func(c *Config) { c.PrivateBucket = "../private" },
		"negative lifetime":         func(c *Config) { c.PresignTTLSeconds = -1 },
		"overlong lifetime":         func(c *Config) { c.PresignTTLSeconds = 604801 },
		"negative timeout":          func(c *Config) { c.RequestTimeoutSeconds = -1 },
	} {
		t.Run(name, func(t *testing.T) {
			config := storageConfig("http://rustfs:9000")
			mutate(&config)
			if _, err := New(config); err == nil {
				t.Fatal("invalid storage configuration accepted")
			} else if strings.Contains(err.Error(), "test-secret") || strings.Contains(err.Error(), "test-access") {
				t.Fatal("configuration error leaked credentials")
			}
		})
	}
}

func TestRustFSStorageFailureAndCancellationAreErrorsWithoutSecretLeakage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `<Error><Code>AccessDenied</Code><Message>test-secret test-access</Message></Error>`)
	}))
	defer server.Close()
	client, err := New(storageConfig(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	for _, operation := range []func() error{
		func() error {
			return client.Put(context.Background(), "go-zero-admin/image.png", "image/png", []byte("image bytes"), "public")
		},
		func() error { return client.Delete(context.Background(), "go-zero-admin/image.png", "private") },
	} {
		err := operation()
		if err == nil {
			t.Fatal("S3 failure reported success")
		}
		if strings.Contains(err.Error(), "test-secret") || strings.Contains(err.Error(), "test-access") {
			t.Fatal("storage error leaked credentials")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := client.Put(ctx, "go-zero-admin/image.png", "image/png", []byte("image bytes"), "public"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
	if err := client.Delete(context.Background(), "key", "unknown"); err == nil {
		t.Fatal("unknown visibility accepted")
	}
	if _, _, err := client.URL(context.Background(), "key", "unknown"); err == nil {
		t.Fatal("unknown URL visibility accepted")
	}
}

func TestRustFSRequestTimeoutBoundsNetworkCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer server.Close()
	config := storageConfig(server.URL)
	config.RequestTimeoutSeconds = 1
	client, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	if err := client.Delete(context.Background(), "go-zero-admin/image.png", "public"); err == nil {
		t.Fatal("timeout reported success")
	}
	if time.Since(started) > 3*time.Second {
		t.Fatal("request timeout did not bound call duration")
	}
}
