package app

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type tokenClaims struct {
	UserID    string `json:"uid"`
	ExpiresAt int64  `json:"exp"`
}

func (s *Service) signToken(userID string) (string, error) {
	payload, err := json.Marshal(tokenClaims{UserID: userID, ExpiresAt: time.Now().Add(30 * 24 * time.Hour).Unix()})
	if err != nil {
		return "", err
	}
	body := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, []byte(s.cfg.TokenSecret))
	_, _ = mac.Write([]byte(body))
	return body + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (s *Service) authRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.Split(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "), ".")
		if len(parts) != 2 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid token"})
			return
		}
		mac := hmac.New(sha256.New, []byte(s.cfg.TokenSecret))
		_, _ = mac.Write([]byte(parts[0]))
		expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
		if subtle.ConstantTimeCompare([]byte(parts[1]), []byte(expected)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		payload, err := base64.RawURLEncoding.DecodeString(parts[0])
		var claims tokenClaims
		if err != nil || json.Unmarshal(payload, &claims) != nil || claims.UserID == "" || claims.ExpiresAt < time.Now().Unix() {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "expired or invalid token"})
			return
		}
		c.Set("userID", claims.UserID)
		c.Next()
	}
}
