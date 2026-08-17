// Package smssender delivers Platform SMS test messages through vendor HTTP APIs.
package smssender

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	smsmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/sms/model"
)

type HTTPSender struct {
	client *http.Client
}

func New() *HTTPSender {
	return &HTTPSender{client: &http.Client{Timeout: 10 * time.Second}}
}

func (s *HTTPSender) Send(ctx context.Context, driver smsmodel.Driver, config map[string]string, phone, code string) (string, error) {
	switch driver {
	case smsmodel.DriverMock:
		return fmt.Sprintf("mock accepted phone=%s code=%s", phone, code), nil
	case smsmodel.DriverAliyun:
		return s.sendAliyun(ctx, config, phone, code)
	case smsmodel.DriverYunpian:
		return s.sendYunpian(ctx, config, phone, code)
	case smsmodel.DriverTwilio:
		return s.sendTwilio(ctx, config, phone, code)
	default:
		return "", smsmodel.ErrInvalid
	}
}

func (s *HTTPSender) sendAliyun(ctx context.Context, config map[string]string, phone, code string) (string, error) {
	keyID := strings.TrimSpace(config["access_key_id"])
	secret := strings.TrimSpace(config["access_key_secret"])
	sign := strings.TrimSpace(config["sign_name"])
	template := strings.TrimSpace(config["template_code"])
	if keyID == "" || secret == "" || sign == "" || template == "" {
		return "", fmt.Errorf("aliyun: access_key_id/secret/sign_name/template_code are required")
	}
	region := strings.TrimSpace(config["region_id"])
	if region == "" {
		region = "cn-hangzhou"
	}
	param, _ := json.Marshal(map[string]string{"code": code})
	params := map[string]string{
		"Format": "JSON", "Version": "2017-05-25", "AccessKeyId": keyID,
		"SignatureMethod": "HMAC-SHA1", "SignatureVersion": "1.0", "SignatureNonce": nonce(),
		"Timestamp": time.Now().UTC().Format("2006-01-02T15:04:05Z"), "RegionId": region,
		"Action": "SendSms", "PhoneNumbers": phone, "SignName": sign, "TemplateCode": template,
		"TemplateParam": string(param),
	}
	params["Signature"] = signRPC(http.MethodGet, params, secret)
	query := url.Values{}
	for key, value := range params {
		query.Set(key, value)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://dysmsapi.aliyuncs.com/?"+query.Encode(), nil)
	if err != nil {
		return "", err
	}
	body, err := s.do(request)
	if err != nil {
		return "", err
	}
	var out struct{ Code, Message, BizId string }
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("aliyun: bad response")
	}
	if out.Code != "OK" {
		return "", fmt.Errorf("aliyun: %s %s", out.Code, out.Message)
	}
	return "aliyun accepted bizId=" + out.BizId, nil
}

func (s *HTTPSender) sendYunpian(ctx context.Context, config map[string]string, phone, code string) (string, error) {
	apiKey := strings.TrimSpace(config["api_key"])
	if apiKey == "" {
		return "", fmt.Errorf("yunpian: api_key is required")
	}
	text := strings.TrimSpace(config["text_template"])
	if text == "" {
		text = "Your verification code is {code}"
	}
	text = strings.ReplaceAll(text, "{code}", code)
	endpoint := strings.TrimSpace(config["endpoint"])
	if endpoint == "" {
		endpoint = "https://sms.yunpian.com/v2/sms/single_send.json"
	}
	form := url.Values{"apikey": {apiKey}, "mobile": {phone}, "text": {text}}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	body, err := s.do(request)
	if err != nil {
		return "", err
	}
	var out struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Sid  int64  `json:"sid"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("yunpian: bad response")
	}
	if out.Code != 0 {
		return "", fmt.Errorf("yunpian: %d %s", out.Code, out.Msg)
	}
	return fmt.Sprintf("yunpian accepted sid=%d", out.Sid), nil
}

func (s *HTTPSender) sendTwilio(ctx context.Context, config map[string]string, phone, code string) (string, error) {
	sid := strings.TrimSpace(config["account_sid"])
	token := strings.TrimSpace(config["auth_token"])
	from := strings.TrimSpace(config["from"])
	if sid == "" || token == "" || from == "" {
		return "", fmt.Errorf("twilio: account_sid/auth_token/from are required")
	}
	text := strings.TrimSpace(config["text_template"])
	if text == "" {
		text = "Your verify code: {code}"
	}
	text = strings.ReplaceAll(text, "{code}", code)
	form := url.Values{"To": {phone}, "From": {from}, "Body": {text}}
	endpoint := "https://api.twilio.com/2010-04-01/Accounts/" + url.PathEscape(sid) + "/Messages.json"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(sid, token)
	body, err := s.do(request)
	if err != nil {
		return "", err
	}
	var out struct {
		SID   string `json:"sid"`
		Error string `json:"message"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("twilio: bad response")
	}
	if out.SID == "" {
		if out.Error == "" {
			out.Error = "send failed"
		}
		return "", fmt.Errorf("twilio: %s", out.Error)
	}
	return "twilio accepted sid=" + out.SID, nil
}

func (s *HTTPSender) do(request *http.Request) ([]byte, error) {
	response, err := s.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return io.ReadAll(io.LimitReader(response.Body, 1<<20))
}

func signRPC(method string, params map[string]string, secret string) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var encoded strings.Builder
	for index, key := range keys {
		if index > 0 {
			encoded.WriteByte('&')
		}
		encoded.WriteString(pctEncode(key))
		encoded.WriteByte('=')
		encoded.WriteString(pctEncode(params[key]))
	}
	stringToSign := method + "&" + pctEncode("/") + "&" + pctEncode(encoded.String())
	mac := hmac.New(sha1.New, []byte(secret+"&"))
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func pctEncode(value string) string {
	encoded := url.QueryEscape(value)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	return strings.ReplaceAll(encoded, "%7E", "~")
}

func nonce() string {
	raw := make([]byte, 16)
	_, _ = rand.Read(raw)
	return hex.EncodeToString(raw)
}
