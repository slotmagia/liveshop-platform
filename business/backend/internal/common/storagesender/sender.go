// Package storagesender writes probe objects through in-process storage drivers.
package storagesender

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	storagemodel "github.com/liveshop-platform/module-platform/internal/biz/capability/storage/model"
)

type Sender struct {
	client *http.Client
	root   string
}

func New() *Sender {
	return &Sender{
		client: &http.Client{Timeout: 30 * time.Second},
		root:   filepath.Join(os.TempDir(), "liveshop-uploads"),
	}
}

func (s *Sender) Put(ctx context.Context, driver storagemodel.Driver, config map[string]string, key string, body []byte) (string, error) {
	key = strings.TrimLeft(filepath.ToSlash(key), "/")
	if key == "" || strings.Contains(key, "..") {
		return "", storagemodel.ErrInvalid
	}
	switch driver {
	case storagemodel.DriverLocal:
		return s.putLocal(key, body)
	case storagemodel.DriverAliyunOSS:
		return s.putOSS(ctx, config, key, body)
	case storagemodel.DriverCloudflareR2:
		return s.putR2(ctx, config, key, body)
	default:
		return "", storagemodel.ErrInvalid
	}
}

func (s *Sender) putLocal(key string, body []byte) (string, error) {
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return "", fmt.Errorf("storage local: mkdir root: %w", err)
	}
	dst := filepath.Join(s.root, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", fmt.Errorf("storage local: mkdir: %w", err)
	}
	if err := os.WriteFile(dst, body, 0o644); err != nil {
		return "", fmt.Errorf("storage local: write: %w", err)
	}
	return "/uploads/" + key, nil
}

func (s *Sender) GetLocal(key string) (storagemodel.Object, error) {
	key = strings.TrimLeft(filepath.ToSlash(key), "/")
	if key == "" || strings.Contains(key, "..") {
		return storagemodel.Object{}, storagemodel.ErrInvalid
	}
	body, err := os.ReadFile(filepath.Join(s.root, filepath.FromSlash(key)))
	if err != nil {
		if os.IsNotExist(err) {
			return storagemodel.Object{}, storagemodel.ErrNotFound
		}
		return storagemodel.Object{}, fmt.Errorf("storage local: read: %w", err)
	}
	contentType := "text/plain; charset=utf-8"
	if !strings.HasSuffix(strings.ToLower(key), ".txt") && !strings.HasSuffix(strings.ToLower(key), ".text") {
		contentType = http.DetectContentType(body)
	}
	return storagemodel.Object{Key: key, ContentType: contentType, Content: string(body)}, nil
}

func (s *Sender) putOSS(ctx context.Context, config map[string]string, key string, body []byte) (string, error) {
	endpoint := strings.TrimRight(strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(config["endpoint"]), "https://"), "http://"), "/")
	bucket := strings.TrimSpace(config["bucket"])
	ak := strings.TrimSpace(config["access_key_id"])
	sk := config["access_key_secret"]
	if endpoint == "" || bucket == "" || ak == "" || sk == "" {
		return "", fmt.Errorf("storage oss: endpoint/bucket/access_key_id/access_key_secret are required")
	}
	host := bucket + "." + endpoint
	base := strings.TrimRight(strings.TrimSpace(config["public_base_url"]), "/")
	if base == "" {
		base = "https://" + host
	}
	const acl = "public-read"
	contentType := "text/plain"
	date := time.Now().UTC().Format(http.TimeFormat)
	stringToSign := strings.Join([]string{http.MethodPut, "", contentType, date, "x-oss-object-acl:" + acl}, "\n") + "\n/" + bucket + "/" + key
	mac := hmac.New(sha1.New, []byte(sk))
	mac.Write([]byte(stringToSign))
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, "https://"+host+"/"+key, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("storage oss: new request: %w", err)
	}
	req.Header.Set("Date", date)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-oss-object-acl", acl)
	req.Header.Set("Authorization", "OSS "+ak+":"+base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	req.ContentLength = int64(len(body))
	if err := doPut(s.client, req, "oss"); err != nil {
		return "", err
	}
	return base + "/" + key, nil
}

func (s *Sender) putR2(ctx context.Context, config map[string]string, key string, body []byte) (string, error) {
	accountID := strings.TrimSpace(config["account_id"])
	bucket := strings.TrimSpace(config["bucket"])
	ak := strings.TrimSpace(config["access_key_id"])
	sk := config["secret_access_key"]
	base := strings.TrimRight(strings.TrimSpace(config["public_base_url"]), "/")
	if accountID == "" || bucket == "" || ak == "" || sk == "" {
		return "", fmt.Errorf("storage r2: account_id/bucket/access_key_id/secret_access_key are required")
	}
	if base == "" {
		return "", fmt.Errorf("storage r2: public_base_url is required")
	}
	host := accountID + ".r2.cloudflarestorage.com"
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	payloadHash := sha256Hex(body)
	canonicalURI := "/" + bucket + "/" + key
	canonicalHeaders := "host:" + host + "\n" + "x-amz-content-sha256:" + payloadHash + "\n" + "x-amz-date:" + amzDate + "\n"
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"
	canonicalRequest := strings.Join([]string{http.MethodPut, canonicalURI, "", canonicalHeaders, signedHeaders, payloadHash}, "\n")
	scope := dateStamp + "/auto/s3/aws4_request"
	stringToSign := strings.Join([]string{"AWS4-HMAC-SHA256", amzDate, scope, sha256Hex([]byte(canonicalRequest))}, "\n")
	signature := hex.EncodeToString(hmacSHA256(sigV4Key(sk, dateStamp, "auto", "s3"), stringToSign))
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, "https://"+host+canonicalURI, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("storage r2: new request: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("x-amz-content-sha256", payloadHash)
	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("Authorization", fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s", ak, scope, signedHeaders, signature))
	req.ContentLength = int64(len(body))
	if err := doPut(s.client, req, "r2"); err != nil {
		return "", err
	}
	return base + "/" + key, nil
}

func doPut(client *http.Client, req *http.Request, driver string) error {
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("storage %s: do request: %w", driver, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("storage %s: put failed status=%d body=%s", driver, resp.StatusCode, string(msg))
	}
	return nil
}

func sha256Hex(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

func sigV4Key(secret, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), dateStamp)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	return hmacSHA256(kService, "aws4_request")
}
