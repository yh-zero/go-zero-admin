package tenantservicelogic

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/zeromicro/go-zero/core/conf"
	"go-zero-admin/application/tenant/rpc/internal/config"
	"go-zero-admin/application/tenant/rpc/internal/model"
	"go-zero-admin/application/tenant/rpc/internal/svc"
	"go-zero-admin/pkg/objectstorage"
)

func TestTenantRuntimeConfigKeepsProxyAndUploadBudgets(t *testing.T) {
	for _, mode := range []string{"host", "compose"} {
		t.Run(mode, func(t *testing.T) {
			root := filepath.Join("..", "..", "..", "..", "..", "..")
			path := filepath.Join(root, "application/tenant/rpc/etc/tenant.yaml")
			if mode == "compose" {
				t.Setenv("GOZERO_EDGE_PROXY_IP", "172.29.0.9")
				path = filepath.Join(root, "docker/etc/tenant-rpc.yaml")
			}
			var c config.Config
			if err := conf.Load(path, &c, conf.UseEnv()); err != nil {
				t.Fatal(err)
			}
			c.Tenant.StateDir = t.TempDir()
			tenant := &model.TenantRegistry{Slug: "config-test", ComposeProject: "gozero-t-config-test"}
			s := &svc.ServiceContext{Config: c}
			if err := renderTenantEnv(s, tenant, objectstorage.Config{RequestTimeoutSeconds: 30}); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(tenantEnvPath(s, tenant))
			if err != nil {
				t.Fatal(err)
			}
			configs := map[string][]byte{}
			for _, line := range strings.Split(string(data), "\n") {
				key, value, ok := strings.Cut(line, "=")
				if !ok || !strings.HasSuffix(key, "_CONFIG_B64") {
					continue
				}
				decoded, err := base64.StdEncoding.DecodeString(value)
				if err != nil {
					t.Fatal(err)
				}
				configs[key] = decoded
			}
			var api struct {
				TrustedProxies []string
				FileRpc        struct{ Timeout int64 }
			}
			var file struct {
				Timeout int64
				Storage struct{ RequestTimeoutSeconds int64 }
			}
			if err := json.Unmarshal(configs["API_CONFIG_B64"], &api); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(configs["FILE_CONFIG_B64"], &file); err != nil {
				t.Fatal(err)
			}
			wantProxy := []string{"172.21.0.2/32"}
			if mode == "compose" {
				wantProxy = []string{"172.29.0.9/32"}
			}
			if !reflect.DeepEqual(api.TrustedProxies, wantProxy) {
				t.Errorf("租户只能信任已配置edge代理，实际=%v，期望=%v", api.TrustedProxies, wantProxy)
			}
			if file.Timeout <= file.Storage.RequestTimeoutSeconds*1000 || api.FileRpc.Timeout <= file.Timeout {
				t.Errorf("租户上传预算不完整：storage=%ds、server=%dms、client=%dms", file.Storage.RequestTimeoutSeconds, file.Timeout, api.FileRpc.Timeout)
			}
		})
	}
}
