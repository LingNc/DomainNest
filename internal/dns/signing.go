package dns

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// percentEncode percent-encodes a string following Aliyun/OSS rules.
// Unreserved characters (A-Z, a-z, 0-9, -, _, ~, .) are kept as-is.
// All other characters are percent-encoded using uppercase hex.
func percentEncode(s string) string {
	hexCount := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if shouldEscape(c) {
			hexCount++
		}
	}
	if hexCount == 0 {
		return s
	}
	t := make([]byte, len(s)+2*hexCount)
	j := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if shouldEscape(c) {
			t[j] = '%'
			t[j+1] = "0123456789ABCDEF"[c>>4]
			t[j+2] = "0123456789ABCDEF"[c&15]
			j += 3
		} else {
			t[j] = c
			j++
		}
	}
	return string(t)
}

func shouldEscape(c byte) bool {
	if 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' || '0' <= c && c <= '9' || c == '_' || c == '-' || c == '~' || c == '.' {
		return false
	}
	return true
}

// hmacSHA256 computes HMAC-SHA256.
func hmacSHA256Bytes(key []byte, content string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(content))
	return mac.Sum(nil)
}

// hashSHA256Hex computes SHA-256 hex digest.
func hashSHA256Hex(content []byte) string {
	h := sha256.New()
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

// --- Baidu Cloud Signing ---

const (
	baiduDateFormat    = "2006-01-02T15:04:05Z"
	baiduExpirePeriod = "1800"
)

// baiduSigner signs an HTTP request for Baidu Cloud BCE API.
func baiduSigner(accessKeyID, accessSecret string, host string, r *http.Request) {
	authStringPrefix := fmt.Sprintf("bce-auth-v1/%s/%s/%s",
		accessKeyID,
		time.Now().UTC().Format(baiduDateFormat),
		baiduExpirePeriod,
	)

	canonicalURI := baiduCanonicalURI(r)
	canonicalQueryString := ""

	canonicalHeaders := fmt.Sprintf("host:%s", host)

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s",
		r.Method,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
	)

	signingKey := hmacSHA256HexStr(accessSecret, authStringPrefix)
	signature := hmacSHA256HexStr(signingKey, canonicalRequest)

	authString := fmt.Sprintf("%s/host/%s", authStringPrefix, signature)
	r.Header.Set("Authorization", authString)
	r.Header.Set("Host", host)
}

func hmacSHA256HexStr(key, message string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

func baiduCanonicalURI(r *http.Request) string {
	uri := r.URL.Path
	if uri == "" {
		uri = "/"
	}
	parts := strings.Split(uri, "/")
	for i, p := range parts {
		parts[i] = percentEncode(p)
	}
	result := strings.Join(parts, "/")
	if !strings.HasSuffix(result, "/") {
		result = strings.TrimRight(result, "/")
	}
	return result
}

// --- TrafficRoute (Volcengine) Signing ---

const (
	trafficRouteVersion = "2018-08-01"
	trafficRouteService = "DNS"
	trafficRouteRegion  = "cn-north-1"
	trafficRouteHost    = "open.volcengineapi.com"
)

// trafficRouteSigner builds a signed HTTP request for Volcengine TrafficRoute.
func trafficRouteSigner(method string, query map[string][]string, header map[string]string, ak, sk, action string, body []byte) (*http.Request, error) {
	request, _ := http.NewRequest(method, "https://"+trafficRouteHost+"/", bytes.NewReader(body))
	urlValues := url.Values{}
	for k, v := range query {
		urlValues[k] = v
	}
	urlValues["Action"] = []string{action}
	urlValues["Version"] = []string{trafficRouteVersion}
	request.URL.RawQuery = urlValues.Encode()
	for k, v := range header {
		request.Header.Set(k, v)
	}

	now := time.Now().UTC()
	xDate := now.Format("20060102T150405Z")
	shortXDate := xDate[:8]
	xContentSha256 := hashSHA256Hex(body)
	contentType := "application/json"

	signedHeadersStr := "content-type;host;x-content-sha256;x-date"
	canonicalRequestStr := strings.Join([]string{
		request.Method,
		"/",
		request.URL.RawQuery,
		strings.Join([]string{
			"content-type:" + contentType,
			"host:" + request.Host,
			"x-content-sha256:" + xContentSha256,
			"x-date:" + xDate,
		}, "\n"),
		"",
		signedHeadersStr,
		xContentSha256,
	}, "\n")

	hashedCanonicalRequest := hashSHA256Hex([]byte(canonicalRequestStr))
	credentialScope := shortXDate + "/" + trafficRouteRegion + "/" + trafficRouteService + "/request"
	stringToSign := "HMAC-SHA256\n" + xDate + "\n" + credentialScope + "\n" + hashedCanonicalRequest

	kDate := hmacSHA256Bytes([]byte(sk), shortXDate)
	kRegion := hmacSHA256Bytes(kDate, trafficRouteRegion)
	kService := hmacSHA256Bytes(kRegion, trafficRouteService)
	kSigning := hmacSHA256Bytes(kService, "request")
	signature := hex.EncodeToString(hmacSHA256Bytes(kSigning, stringToSign))

	authStr := fmt.Sprintf("HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		ak, credentialScope, signedHeadersStr, signature)

	request.Header.Set("Host", trafficRouteHost)
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("X-Date", xDate)
	request.Header.Set("X-Content-Sha256", xContentSha256)
	request.Header.Set("Authorization", authStr)

	return request, nil
}

// --- Tencent Cloud Signing (EdgeOne/DNSPod) ---

// tencentCloudSigner signs a request for Tencent Cloud API (TC3-HMAC-SHA256).
func tencentCloudSigner(secretId, secretKey, action, payload, service string, r *http.Request) {
	algorithm := "TC3-HMAC-SHA256"
	host := service + ".tencentcloudapi.com"
	timestamp := time.Now().Unix()
	timestampStr := strconv.FormatInt(timestamp, 10)

	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")

	canonicalHeaders := "content-type:application/json\nhost:" + host + "\nx-tc-action:" + strings.ToLower(action) + "\n"
	signedHeaders := "content-type;host;x-tc-action"
	hashedPayload := hashSHA256Hex([]byte(payload))
	canonicalRequest := fmt.Sprintf("POST\n/\n\n%s\n%s\n%s", canonicalHeaders, signedHeaders, hashedPayload)

	credentialScope := date + "/" + service + "/tc3_request"
	hashedCanonicalRequest := hashSHA256Hex([]byte(canonicalRequest))
	stringToSign := algorithm + "\n" + timestampStr + "\n" + credentialScope + "\n" + hashedCanonicalRequest

	secretDate := hmacSHA256Bytes([]byte("TC3"+secretKey), date)
	secretService := hmacSHA256Bytes(secretDate, service)
	secretSigning := hmacSHA256Bytes(secretService, "tc3_request")
	signature := hex.EncodeToString(hmacSHA256Bytes(secretSigning, stringToSign))

	r.Header.Set("Authorization", algorithm+" Credential="+secretId+"/"+credentialScope+", SignedHeaders="+signedHeaders+", Signature="+signature)
	r.Header.Set("Host", host)
	r.Header.Set("X-TC-Action", action)
	r.Header.Set("X-TC-Timestamp", timestampStr)
	r.Header.Set("Content-Type", "application/json")
}

// --- Aliyun/Alibaba Cloud Signing ---

var signMethodMap = map[string]func() hash.Hash{
	"HMAC-SHA1":   sha1.New,
	"HMAC-SHA256": sha256.New,
}

// aliyunSigner signs a request for Alibaba Cloud APIs (Aliyun DNS, ESA, etc.).
func aliyunSigner(accessKeyID, accessSecret string, params *url.Values, httpMethod string, apiVersion string) {
	params.Set("SignatureMethod", "HMAC-SHA1")
	params.Set("SignatureNonce", strconv.FormatInt(time.Now().UnixNano(), 10))
	params.Set("AccessKeyId", accessKeyID)
	params.Set("SignatureVersion", "1.0")
	params.Set("Timestamp", time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	params.Set("Format", "JSON")
	params.Set("Version", apiVersion)
	params.Set("Signature", aliyunHmacSignToB64("HMAC-SHA1", httpMethod, accessSecret, *params))
}

func aliyunHmacSignToB64(signMethod, httpMethod, appKeySecret string, params url.Values) string {
	return base64.StdEncoding.EncodeToString(aliyunHmacSign(signMethod, httpMethod, appKeySecret, params))
}

func aliyunHmacSign(signMethod, httpMethod, appKeySecret string, params url.Values) []byte {
	key := []byte(appKeySecret + "&")
	newHash, ok := signMethodMap[signMethod]
	if !ok {
		newHash = sha1.New
	}
	mac := hmac.New(newHash, key)
	mac.Write([]byte(aliyunMakeDataToSign(httpMethod, params)))
	return mac.Sum(nil)
}

func aliyunMakeDataToSign(httpMethod string, params url.Values) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var canonicalizedQueryString string
	for i, k := range keys {
		if i > 0 {
			canonicalizedQueryString += "&"
		}
		canonicalizedQueryString += percentEncode(k) + "=" + percentEncode(params.Get(k))
	}

	return httpMethod + "&" + percentEncode("/") + "&" + percentEncode(canonicalizedQueryString)
}
