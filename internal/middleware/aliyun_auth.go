package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/url"
	"sync"
	"time"

	"domainnest/internal/dns"
	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
)

// nonceCache prevents signature replay attacks.
type nonceCache struct {
	mu     sync.Mutex
	nonces map[string]time.Time
}

var nc = &nonceCache{nonces: make(map[string]time.Time)}

func (c *nonceCache) seen(nonce string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Cleanup old entries (older than 10 minutes)
	now := time.Now()
	for k, t := range c.nonces {
		if now.Sub(t) > 10*time.Minute {
			delete(c.nonces, k)
		}
	}
	if _, ok := c.nonces[nonce]; ok {
		return true
	}
	c.nonces[nonce] = now
	return false
}

// AliyunAuth verifies Aliyun DNS API HMAC-SHA1 signatures.
func AliyunAuth(ramTokenSvc *service.RAMTokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var params url.Values
		if c.Request.Method == "POST" {
			c.Request.ParseForm()
			params = c.Request.Form
		} else {
			params = c.Request.URL.Query()
		}

		accessKeyID := params.Get("AccessKeyId")
		signature := params.Get("Signature")
		timestamp := params.Get("Timestamp")
		nonce := params.Get("SignatureNonce")

		if accessKeyID == "" || signature == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"RequestId": genRequestID(),
				"HostId":    "alidns",
				"Code":      "MissingParameter",
				"Message":   "The input parameter AccessKeyId or Signature that is mandatory for processing this request is not supplied.",
			})
			c.Abort()
			return
		}

		// Lookup token
		token, err := ramTokenSvc.LookupByAccessKeyID(accessKeyID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"RequestId": genRequestID(),
				"HostId":    "alidns",
				"Code":      "InvalidAccessKeyId.NotFound",
				"Message":   "The specified AccessKeyId does not exist.",
			})
			c.Abort()
			return
		}

		// Check timestamp freshness (±15 minutes)
		ts, err := time.Parse(time.RFC3339, timestamp)
		if err != nil {
			ts, err = time.Parse("2006-01-02T15:04:05Z", timestamp)
		}
		if err != nil || time.Since(ts).Abs() > 15*time.Minute {
			c.JSON(http.StatusBadRequest, gin.H{
				"RequestId": genRequestID(),
				"HostId":    "alidns",
				"Code":      "InvalidTimeStamp.Expired",
				"Message":   "The request timestamp has expired.",
			})
			c.Abort()
			return
		}

		// Check nonce replay
		if nc.seen(nonce) {
			c.JSON(http.StatusBadRequest, gin.H{
				"RequestId": genRequestID(),
				"HostId":    "alidns",
				"Code":      "SignatureNonceUsed",
				"Message":   "The specified signature nonce has been used.",
			})
			c.Abort()
			return
		}

		// Verify signature
		params.Del("Signature")
		expected := dns.VerifyAliyunSignature("GET", token.AccessKeySecret, params)
		if signature != expected {
			c.JSON(http.StatusBadRequest, gin.H{
				"RequestId": genRequestID(),
				"HostId":    "alidns",
				"Code":      "SignatureDoesNotMatch",
				"Message":   "The specified signature is not matched with our calculation.",
			})
			c.Abort()
			return
		}

		c.Set("user_id", token.UserID)
		c.Set("ram_token_id", token.ID)
		c.Next()
	}
}

func genRequestID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}