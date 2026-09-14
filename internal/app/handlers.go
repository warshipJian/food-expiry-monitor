package app

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const dateLayout = "2006-01-02"

func (s *Service) listFoods(c *gin.Context) {
	status := c.DefaultQuery("status", "active")
	if status != "active" && status != "consumed" && status != "discarded" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
		return
	}
	familyID, err := s.familyIDForUser(userID(c))
	if err != nil && err != sql.ErrNoRows {
		internalError(c, scopeError(err))
		return
	}
	scope, scopeArgs := familyScopeClause(userID(c), familyID)
	query := `SELECT id,name,barcode,category,storage_location,quantity,unit,expiry_date,status,created_at,updated_at FROM foods WHERE ` + scope + ` AND status=?`
	args := append(scopeArgs, status)
	if location := c.Query("location"); location != "" {
		query += " AND storage_location=?"
		args = append(args, location)
	}
	query += " ORDER BY expiry_date ASC"
	rows, err := s.db.Query(query, args...)
	if err != nil {
		internalError(c, err)
		return
	}
	defer rows.Close()
	foods := make([]Food, 0)
	for rows.Next() {
		food, err := scanFood(rows)
		if err != nil {
			internalError(c, err)
			return
		}
		foods = append(foods, food)
	}
	if err := rows.Err(); err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": foods})
}

func (s *Service) createFood(c *gin.Context) {
	var input foodInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid food", "details": err.Error()})
		return
	}
	expiry, err := time.Parse(dateLayout, input.ExpiryDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "expiryDate must be YYYY-MM-DD"})
		return
	}
	normalizeFood(&input)
	id := newID()
	familyID, familyErr := s.familyIDForUser(userID(c))
	if familyErr != nil && familyErr != sql.ErrNoRows {
		internalError(c, scopeError(familyErr))
		return
	}
	_, err = s.db.Exec(`INSERT INTO foods (id,user_id,family_id,name,barcode,category,storage_location,quantity,unit,expiry_date) VALUES (?,?,?,?,?,?,?,?,?,?)`, id, userID(c), nullableFamilyID(familyID), input.Name, input.Barcode, input.Category, input.StorageLocation, input.Quantity, input.Unit, expiry.Format(dateLayout))
	if err != nil {
		internalError(c, err)
		return
	}
	s.respondFood(c, id, http.StatusCreated)
}

func (s *Service) getFood(c *gin.Context) { s.respondFood(c, c.Param("id"), http.StatusOK) }

func (s *Service) respondFood(c *gin.Context, id string, successStatus int) {
	food, err := s.findFood(userID(c), id)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "food not found"})
		return
	}
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(successStatus, food)
}

func (s *Service) updateFood(c *gin.Context) {
	var input foodInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid food", "details": err.Error()})
		return
	}
	expiry, err := time.Parse(dateLayout, input.ExpiryDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "expiryDate must be YYYY-MM-DD"})
		return
	}
	normalizeFood(&input)
	scope, scopeArgs, err := s.foodScope(userID(c))
	if err != nil {
		internalError(c, err)
		return
	}
	args := []any{input.Name, input.Barcode, input.Category, input.StorageLocation, input.Quantity, input.Unit, expiry.Format(dateLayout), c.Param("id")}
	args = append(args, scopeArgs...)
	result, err := s.db.Exec(`UPDATE foods SET name=?,barcode=?,category=?,storage_location=?,quantity=?,unit=?,expiry_date=?,updated_at=CURRENT_TIMESTAMP WHERE id=? AND `+scope+` AND status='active'`, args...)
	if err != nil {
		internalError(c, err)
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "active food not found"})
		return
	}
	_, _ = s.db.Exec("UPDATE reminder_jobs SET status='cancelled' WHERE food_id=? AND status='pending'", c.Param("id"))
	s.respondFood(c, c.Param("id"), http.StatusOK)
}

func (s *Service) consumeFood(c *gin.Context) { s.changeFoodStatus(c, "consumed") }
func (s *Service) discardFood(c *gin.Context) { s.changeFoodStatus(c, "discarded") }
func (s *Service) changeFoodStatus(c *gin.Context, status string) {
	scope, scopeArgs, err := s.foodScope(userID(c))
	if err != nil {
		internalError(c, err)
		return
	}
	args := []any{status, c.Param("id")}
	args = append(args, scopeArgs...)
	result, err := s.db.Exec("UPDATE foods SET status=?,updated_at=CURRENT_TIMESTAMP WHERE id=? AND "+scope+" AND status='active'", args...)
	if err != nil {
		internalError(c, err)
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "active food not found"})
		return
	}
	_, _ = s.db.Exec("UPDATE reminder_jobs SET status='cancelled' WHERE food_id=? AND status='pending'", c.Param("id"))
	c.Status(http.StatusNoContent)
}

func (s *Service) getDashboard(c *gin.Context) {
	var result dashboard
	scope, args, err := s.foodScope(userID(c))
	if err != nil {
		internalError(c, err)
		return
	}
	err = s.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(CASE WHEN expiry_date >= date('now') AND expiry_date <= date('now','+3 days') THEN 1 ELSE 0 END),0), COALESCE(SUM(CASE WHEN expiry_date < date('now') THEN 1 ELSE 0 END),0) FROM foods WHERE `+scope+` AND status='active'`, args...).Scan(&result.Active, &result.Expiring, &result.Expired)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

type reminderInput struct {
	DaysBefore int `json:"daysBefore" binding:"gte=0,lte=60"`
}

func (s *Service) createReminder(c *gin.Context) {
	var input reminderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "daysBefore must be between 0 and 60"})
		return
	}
	food, err := s.findFood(userID(c), c.Param("id"))
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "food not found"})
		return
	}
	if err != nil {
		internalError(c, err)
		return
	}
	if food.Status != "active" {
		c.JSON(http.StatusConflict, gin.H{"error": "food is no longer active"})
		return
	}
	remindAt := time.Date(food.ExpiryDate.Year(), food.ExpiryDate.Month(), food.ExpiryDate.Day()-input.DaysBefore, s.cfg.ReminderCronHour, 0, 0, 0, time.Local)
	if remindAt.Before(time.Now()) {
		remindAt = time.Now()
	}
	_, _ = s.db.Exec("UPDATE reminder_jobs SET status='cancelled' WHERE food_id=? AND status='pending'", food.ID)
	_, err = s.db.Exec(`INSERT INTO reminder_jobs (id,food_id,user_id,remind_at,template_id) VALUES (?,?,?,?,?)`, newID(), food.ID, userID(c), remindAt, s.cfg.WeChatTemplateID)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"foodId": food.ID, "remindAt": remindAt, "status": "pending"})
}

func (s *Service) findFood(uid, id string) (Food, error) {
	scope, args, err := s.foodScope(uid)
	if err != nil {
		return Food{}, err
	}
	args = append([]any{id}, args...)
	return scanFood(s.db.QueryRow(`SELECT id,name,barcode,category,storage_location,quantity,unit,expiry_date,status,created_at,updated_at FROM foods WHERE id=? AND `+scope, args...))
}

func (s *Service) foodScope(uid string) (string, []any, error) {
	familyID, err := s.familyIDForUser(uid)
	if err != nil && err != sql.ErrNoRows {
		return "", nil, scopeError(err)
	}
	scope, args := familyScopeClause(uid, familyID)
	return scope, args, nil
}

func nullableFamilyID(familyID string) any {
	if familyID == "" {
		return nil
	}
	return familyID
}

type scanner interface{ Scan(...any) error }

func scanFood(row scanner) (Food, error) {
	var food Food
	err := row.Scan(&food.ID, &food.Name, &food.Barcode, &food.Category, &food.StorageLocation, &food.Quantity, &food.Unit, &food.ExpiryDate, &food.Status, &food.CreatedAt, &food.UpdatedAt)
	return food, err
}
func normalizeFood(input *foodInput) {
	input.Name = strings.TrimSpace(input.Name)
	input.Barcode = strings.TrimSpace(input.Barcode)
	if input.Category == "" {
		input.Category = "other"
	}
	if input.StorageLocation == "" {
		input.StorageLocation = "fridge"
	}
	if input.Quantity == 0 {
		input.Quantity = 1
	}
	if input.Unit == "" {
		input.Unit = "item"
	}
}
func newID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}
func internalError(c *gin.Context, err error) {
	c.Error(fmt.Errorf("internal error: %w", err))
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}

type wechatLoginInput struct {
	Code string `json:"code" binding:"required"`
}

type devLoginInput struct {
	DeviceID string `json:"deviceId" binding:"required,max=64"`
}

// devLogin exists only outside production so the Mini Program UI can be
// developed before WeChat credentials are available.
func (s *Service) devLogin(c *gin.Context) {
	var input devLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "deviceId is required"})
		return
	}
	openid := "dev:" + strings.TrimSpace(input.DeviceID)
	var uid string
	err := s.db.QueryRow("SELECT id FROM users WHERE openid=?", openid).Scan(&uid)
	if err == sql.ErrNoRows {
		uid = newID()
		_, err = s.db.Exec("INSERT INTO users (id,openid,nickname) VALUES (?,?,?)", uid, openid, "Local Developer")
	}
	if err != nil {
		internalError(c, err)
		return
	}
	token, err := s.signToken(uid)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "expiresIn": 2592000})
}

func (s *Service) wechatLogin(c *gin.Context) {
	var input wechatLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}
	if s.cfg.WeChatAppID == "" || s.cfg.WeChatAppSecret == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "WeChat login is not configured"})
		return
	}
	endpoint := "https://api.weixin.qq.com/sns/jscode2session?appid=" + url.QueryEscape(s.cfg.WeChatAppID) + "&secret=" + url.QueryEscape(s.cfg.WeChatAppSecret) + "&js_code=" + url.QueryEscape(input.Code) + "&grant_type=authorization_code"
	response, err := http.Get(endpoint)
	if err != nil {
		internalError(c, err)
		return
	}
	defer response.Body.Close()
	var payload struct {
		OpenID  string `json:"openid"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil || payload.ErrCode != 0 || payload.OpenID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "WeChat login failed"})
		return
	}
	var uid string
	err = s.db.QueryRow("SELECT id FROM users WHERE openid=?", payload.OpenID).Scan(&uid)
	if err == sql.ErrNoRows {
		uid = newID()
		_, err = s.db.Exec("INSERT INTO users (id,openid) VALUES (?,?)", uid, payload.OpenID)
	}
	if err != nil {
		internalError(c, err)
		return
	}
	token, err := s.signToken(uid)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "expiresIn": 2592000})
}
