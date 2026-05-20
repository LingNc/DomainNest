package ws

import (
	"net/http"

	"domainnest/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// HandleUpgrade returns a gin.HandlerFunc that authenticates the
// request via a JWT passed in the "token" query parameter, upgrades
// the HTTP connection to WebSocket, and starts the client pumps.
func HandleUpgrade(hub *Hub, jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.Query("token")
		if tokenStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "missing token query parameter"})
			return
		}

		claims := &middleware.JWTClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid or expired token"})
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}

		client := NewClient(hub, conn, claims.UserID)
		hub.Register(client)

		go client.WritePump()
		go client.ReadPump()
	}
}
