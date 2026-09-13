package app

import (
	"database/sql"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type Service struct {
	db                 *sql.DB
	cfg                Config
	httpClient         *http.Client
	accessToken        string
	accessTokenExpires time.Time
	accessTokenMu      sync.Mutex
}

func NewService(db *sql.DB, cfg Config) *Service {
	return &Service{db: db, cfg: cfg, httpClient: &http.Client{Timeout: 10 * time.Second}}
}

func (s *Service) Router() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.Static("/uploads", s.cfg.UploadDir)
	r.POST("/v1/auth/wechat/login", s.wechatLogin)
	if s.cfg.Environment != "production" {
		r.POST("/v1/auth/dev/login", s.devLogin)
	}
	v1 := r.Group("/v1")
	v1.Use(s.authRequired())
	v1.GET("/dashboard", s.getDashboard)
	v1.GET("/profile", s.getProfile)
	v1.PATCH("/profile", s.updateProfile)
	v1.POST("/profile/avatar", s.uploadAvatar)
	v1.GET("/foods", s.listFoods)
	v1.POST("/foods", s.createFood)
	v1.GET("/foods/:id", s.getFood)
	v1.PATCH("/foods/:id", s.updateFood)
	v1.POST("/foods/:id/consume", s.consumeFood)
	v1.POST("/foods/:id/discard", s.discardFood)
	v1.POST("/foods/:id/reminder", s.createReminder)
	return r
}

func userID(c *gin.Context) string { value, _ := c.Get("userID"); return value.(string) }
