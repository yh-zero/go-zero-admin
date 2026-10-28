// Package objectstorage provides the RustFS S3 storage used by the business RPC.
package objectstorage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Config belongs only to the RPC process. Zero configuration disables storage;
// a partially supplied configuration is a startup error.
type Config struct {
	Endpoint              string `json:",optional"`
	ExternalEndpoint      string `json:",optional"`
	Region                string `json:",optional"`
	AccessKeyID           string `json:",optional"`
	SecretAccessKey       string `json:",optional"`
	PublicBucket          string `json:",optional"`
	PrivateBucket         string `json:",optional"`
	PresignTTLSeconds     int64  `json:",optional"`
	RequestTimeoutSeconds int64  `json:",optional"`
}

var ErrNotConfigured = errors.New("文件存储服务未配置")

type Client struct {
	internal *s3.Client
	external *s3.PresignClient
	config   Config
	timeout  time.Duration
}

var bucketPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`)
var regionPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// New validates configuration without making network requests or consulting
// environment credentials, instance metadata or shared AWS configuration.
func New(config Config) (*Client, error) {
	if config == (Config{}) {
		return nil, nil
	}
	for _, field := range []struct{ name, value string }{
		{"Endpoint", config.Endpoint}, {"ExternalEndpoint", config.ExternalEndpoint},
		{"Region", config.Region}, {"AccessKeyID", config.AccessKeyID},
		{"SecretAccessKey", config.SecretAccessKey}, {"PublicBucket", config.PublicBucket},
		{"PrivateBucket", config.PrivateBucket},
	} {
		if strings.TrimSpace(field.value) == "" {
			return nil, fmt.Errorf("invalid Storage config: %s is required", field.name)
		}
	}
	for _, field := range []struct{ name, value string }{{"Endpoint", config.Endpoint}, {"ExternalEndpoint", config.ExternalEndpoint}} {
		u, err := url.Parse(field.value)
		if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
			return nil, fmt.Errorf("invalid Storage config: %s must be an HTTP(S) origin", field.name)
		}
	}
	if !regionPattern.MatchString(config.Region) {
		return nil, errors.New("invalid Storage config: Region is invalid")
	}
	for _, bucket := range []string{config.PublicBucket, config.PrivateBucket} {
		if !bucketPattern.MatchString(bucket) || strings.Contains(bucket, "..") || strings.Contains(bucket, ".-") || strings.Contains(bucket, "-.") || net.ParseIP(bucket) != nil {
			return nil, errors.New("invalid Storage config: bucket name is invalid")
		}
	}
	if config.PublicBucket == config.PrivateBucket {
		return nil, errors.New("invalid Storage config: PublicBucket and PrivateBucket must differ")
	}
	if config.PresignTTLSeconds < 0 || config.PresignTTLSeconds > 604800 {
		return nil, errors.New("invalid Storage config: PresignTTLSeconds must be between 1 and 604800, or 0 for the default")
	}
	if config.RequestTimeoutSeconds < 0 || config.RequestTimeoutSeconds > int64((1<<63-1)/time.Second) {
		return nil, errors.New("invalid Storage config: RequestTimeoutSeconds is invalid")
	}
	if config.PresignTTLSeconds == 0 {
		config.PresignTTLSeconds = 300
	}
	if config.RequestTimeoutSeconds == 0 {
		config.RequestTimeoutSeconds = 10
	}
	config.Endpoint = strings.TrimRight(config.Endpoint, "/")
	config.ExternalEndpoint = strings.TrimRight(config.ExternalEndpoint, "/")
	timeout := time.Duration(config.RequestTimeoutSeconds) * time.Second
	awsConfig := aws.Config{
		Region:                     config.Region,
		Credentials:                credentials.NewStaticCredentialsProvider(config.AccessKeyID, config.SecretAccessKey, ""),
		HTTPClient:                 &http.Client{Timeout: timeout},
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
	}
	newS3 := func(endpoint string) *s3.Client {
		return s3.NewFromConfig(awsConfig, func(options *s3.Options) {
			options.BaseEndpoint = aws.String(endpoint)
			options.UsePathStyle = true
		})
	}
	return &Client{internal: newS3(config.Endpoint), external: s3.NewPresignClient(newS3(config.ExternalEndpoint)), config: config, timeout: timeout}, nil
}

func (c *Client) bucket(visibility string) (string, error) {
	if c == nil {
		return "", ErrNotConfigured
	}
	switch visibility {
	case "public":
		return c.config.PublicBucket, nil
	case "private":
		return c.config.PrivateBucket, nil
	default:
		return "", errors.New("invalid object visibility")
	}
}

func (c *Client) Put(ctx context.Context, key, mime string, data []byte, visibility string) error {
	bucket, err := c.bucket(visibility)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	_, err = c.internal.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(bucket), Key: aws.String(key), Body: bytes.NewReader(data), ContentType: aws.String(mime)})
	return storageError(ctx, "upload", err)
}

func (c *Client) Delete(ctx context.Context, key, visibility string) error {
	bucket, err := c.bucket(visibility)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	_, err = c.internal.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	return storageError(ctx, "delete", err)
}

// URL uses the browser-facing endpoint before signing. Rewriting a signed
// internal hostname afterwards would invalidate the S3 signature.
func (c *Client) URL(ctx context.Context, key, visibility string) (string, int64, error) {
	bucket, err := c.bucket(visibility)
	if err != nil {
		return "", 0, err
	}
	if visibility == "public" {
		u, _ := url.Parse(c.config.ExternalEndpoint)
		u.Path = "/" + bucket + "/" + key
		return u.String(), 0, nil
	}
	request, err := c.external.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)}, func(options *s3.PresignOptions) {
		options.Expires = time.Duration(c.config.PresignTTLSeconds) * time.Second
	})
	if err != nil {
		return "", 0, storageError(ctx, "sign URL", err)
	}
	return request.URL, c.config.PresignTTLSeconds, nil
}

// S3 error bodies can reflect request credentials. Preserve cancellation while
// returning only controlled messages to callers and logs.
func storageError(ctx context.Context, operation string, err error) error {
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return fmt.Errorf("object storage %s failed: %w", operation, ctx.Err())
	}
	return fmt.Errorf("object storage %s failed", operation)
}
