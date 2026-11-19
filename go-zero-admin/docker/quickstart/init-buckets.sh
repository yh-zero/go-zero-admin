#!/bin/sh
# 文件服务使用两个固定桶。保留已有对象，只给公开桶授予匿名读取对象的权限。
set -eu
storage=http://rustfs:9000
s3() {
    curl -fsS --aws-sigv4 aws:amz:us-east-1:s3 --user "$RUSTFS_ACCESS_KEY:$RUSTFS_SECRET_KEY" "$@"
}
for bucket in gozero-public gozero-private; do
    if ! s3 --head "$storage/$bucket" >/dev/null 2>&1; then
        s3 -X PUT "$storage/$bucket"
    fi
done
s3 -X PUT -H 'Content-Type: application/json' \
    --data '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":"*"},"Action":["s3:GetObject"],"Resource":["arn:aws:s3:::gozero-public/*"]}]}' \
    "$storage/gozero-public?policy"
