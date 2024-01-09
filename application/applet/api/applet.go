package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"net/http"
	"os"

	"go-zero-admin/application/applet/api/internal/config"
	"go-zero-admin/application/applet/api/internal/handler"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/pkg/result"
	"go-zero-admin/pkg/result/xerr"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/applet-api.yaml", "the config file")

// var OssEnv = os.Getenv("AliYunOss")
var OssEnv = os.Getenv("AliYunOss_go-zero-admin")

type AliYunOss struct {
	Endpoint        string
	AccessKeyId     string
	AccessKeySecret string
	BucketName      string
}

func getEnv(c *config.Config, o *AliYunOss) {
	c.Oss.AccessKeyId = o.AccessKeyId
	c.Oss.AccessKeySecret = o.AccessKeySecret
	c.Oss.BucketName = o.BucketName
	c.Oss.Endpoint = o.Endpoint
}

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 把阿里云的值存到本地环境中
	var aliYunOssEnv AliYunOss
	err := json.Unmarshal([]byte(OssEnv), &aliYunOssEnv)
	if err != nil {
		logx.Errorf("Obtain local environment Err: %v", err)
		panic(err)
	}
	getEnv(&c, &aliYunOssEnv)

	server := rest.MustNewServer(c.RestConf, rest.WithUnauthorizedCallback(func(w http.ResponseWriter, r *http.Request, err error) {
		result.HttpResult(r, w, "", xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR))
	}))
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
