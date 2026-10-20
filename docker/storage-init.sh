#!/bin/sh
set -eu
umask 077

: "${RUSTFS_ACCESS_KEY:?root access key required}"
: "${RUSTFS_SECRET_KEY:?root secret required}"
: "${STORAGE_ACCESS_KEY_ID:?application access key required}"
: "${STORAGE_SECRET_ACCESS_KEY:?application secret required}"
: "${STORAGE_PUBLIC_BUCKET:?public bucket required}"
: "${STORAGE_PRIVATE_BUCKET:?private bucket required}"
: "${STORAGE_CORS_ORIGINS:?frontend origins required}"
STORAGE_ENDPOINT=${STORAGE_ENDPOINT:-http://rustfs:9000}
AWS_DEFAULT_REGION=${STORAGE_REGION:-us-east-1}
export AWS_DEFAULT_REGION AWS_ACCESS_KEY_ID="$RUSTFS_ACCESS_KEY" AWS_SECRET_ACCESS_KEY="$RUSTFS_SECRET_KEY"
export AWS_EC2_METADATA_DISABLED=true AWS_PAGER=''

if [ "$STORAGE_PUBLIC_BUCKET" = "$STORAGE_PRIVATE_BUCKET" ] || [ "$RUSTFS_ACCESS_KEY" = "$STORAGE_ACCESS_KEY_ID" ]; then
    echo 'Public/private buckets and root/application identities must be distinct' >&2
    exit 1
fi
case "$STORAGE_ACCESS_KEY_ID" in *[!a-zA-Z0-9_-]*|'') echo 'Invalid application access key' >&2; exit 1;; esac
for bucket in "$STORAGE_PUBLIC_BUCKET" "$STORAGE_PRIVATE_BUCKET"; do
    case "$bucket" in *[!a-z0-9.-]*|'') echo 'Invalid bucket name' >&2; exit 1;; esac
done

aws_s3() { aws --endpoint-url "$STORAGE_ENDPOINT" s3api "$@"; }
admin_put() {
    curl --silent --show-error --fail --connect-timeout 5 --max-time 30 \
        --aws-sigv4 "aws:amz:$AWS_DEFAULT_REGION:s3" \
        --user "$RUSTFS_ACCESS_KEY:$RUSTFS_SECRET_KEY" \
        -H 'Content-Type: application/json' -X PUT --data-binary "@$2" \
        "$STORAGE_ENDPOINT/rustfs/admin/v3/$1" >/dev/null
}

# Head/create only: reruns must never remove buckets or existing objects.
for bucket in "$STORAGE_PUBLIC_BUCKET" "$STORAGE_PRIVATE_BUCKET"; do
    if ! aws_s3 head-bucket --bucket "$bucket" >/dev/null 2>&1; then
        aws_s3 create-bucket --bucket "$bucket" >/dev/null
    fi
done

jq -n --arg bucket "$STORAGE_PUBLIC_BUCKET" '{Version:"2012-10-17",Statement:[{Sid:"PublicImages",Effect:"Allow",Principal:"*",Action:["s3:GetObject"],Resource:[("arn:aws:s3:::"+$bucket+"/go-zero-admin/*")]}]}' >/tmp/public-policy.json
aws_s3 put-public-access-block --bucket "$STORAGE_PUBLIC_BUCKET" --public-access-block-configuration 'BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=false,RestrictPublicBuckets=false'
aws_s3 put-bucket-policy --bucket "$STORAGE_PUBLIC_BUCKET" --policy file:///tmp/public-policy.json
aws_s3 delete-bucket-policy --bucket "$STORAGE_PRIVATE_BUCKET"
aws_s3 put-public-access-block --bucket "$STORAGE_PRIVATE_BUCKET" --public-access-block-configuration 'BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true'

# Browser download uses fetch without application JWT; only explicit UI origins may read.
jq -n --arg origins "$STORAGE_CORS_ORIGINS" '{CORSRules:[{AllowedOrigins:($origins|split(",")|map(gsub("^\\s+|\\s+$";"") )|map(select(length>0))),AllowedMethods:["GET","HEAD"],AllowedHeaders:["*"],ExposeHeaders:["Content-Length","Content-Type","Content-Disposition","ETag"],MaxAgeSeconds:300}]}' >/tmp/cors.json
for bucket in "$STORAGE_PUBLIC_BUCKET" "$STORAGE_PRIVATE_BUCKET"; do
    aws_s3 put-bucket-cors --bucket "$bucket" --cors-configuration file:///tmp/cors.json
done

# Provision a dedicated identity. The RPC never receives root/admin credentials.
jq -n --arg secret "$STORAGE_SECRET_ACCESS_KEY" '{secretKey:$secret,status:"enabled"}' >/tmp/user.json
admin_put "add-user?accessKey=$STORAGE_ACCESS_KEY_ID" /tmp/user.json
jq -n --arg public "$STORAGE_PUBLIC_BUCKET" --arg private "$STORAGE_PRIVATE_BUCKET" '{Version:"2012-10-17",Statement:[{Effect:"Allow",Action:["s3:ListBucket","s3:GetBucketLocation"],Resource:[("arn:aws:s3:::"+$public),("arn:aws:s3:::"+$private)]},{Effect:"Allow",Action:["s3:GetObject","s3:PutObject","s3:DeleteObject"],Resource:[("arn:aws:s3:::"+$public+"/go-zero-admin/*"),("arn:aws:s3:::"+$private+"/go-zero-admin/*")]}]}' >/tmp/application-policy.json
admin_put 'add-canned-policy?name=gozero-files' /tmp/application-policy.json
printf '{}' >/tmp/empty.json
admin_put "set-user-or-group-policy?policyName=gozero-files&userOrGroup=$STORAGE_ACCESS_KEY_ID&isGroup=false" /tmp/empty.json

export AWS_ACCESS_KEY_ID="$STORAGE_ACCESS_KEY_ID" AWS_SECRET_ACCESS_KEY="$STORAGE_SECRET_ACCESS_KEY"
aws_s3 head-bucket --bucket "$STORAGE_PUBLIC_BUCKET" >/dev/null
aws_s3 head-bucket --bucket "$STORAGE_PRIVATE_BUCKET" >/dev/null

# Fixed demo keys only. Conditional writes preserve any existing object,
# including one created between concurrent initializer runs.
seed_demo_file() {
    demo_bucket=$1
    demo_key="go-zero-admin/00000000-0000-4000-8000-000000000$2.png"
    if aws_s3 put-object --bucket "$demo_bucket" --key "$demo_key" \
        --body "/opt/gozero/demo-files/$3" --content-type image/png \
        --if-none-match '*' >/dev/null 2>/tmp/demo-put.err; then
        echo "Created demo object: $demo_bucket/$demo_key"
    elif grep -q '(PreconditionFailed)' /tmp/demo-put.err; then
        echo "Preserved existing demo object: $demo_bucket/$demo_key"
    else
        cat /tmp/demo-put.err >&2
        exit 1
    fi
}
seed_demo_file "$STORAGE_PUBLIC_BUCKET" 101 public-landscape.png
seed_demo_file "$STORAGE_PUBLIC_BUCKET" 102 public-shapes.png
seed_demo_file "$STORAGE_PRIVATE_BUCKET" 103 private-lock.png
echo 'RustFS buckets, access policies, CORS, application identity and demo files are ready.'
