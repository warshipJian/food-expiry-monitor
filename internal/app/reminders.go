package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type reminderDelivery struct {
	ID       string
	FoodID   string
	OpenID   string
	Name     string
	Quantity float64
	Expiry   time.Time
}

// RunReminderScheduler checks due jobs once at startup and then every minute.
func (s *Service) RunReminderScheduler(ctx context.Context) {
	s.dispatchDueReminders(ctx)
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.dispatchDueReminders(ctx)
		}
	}
}

func (s *Service) dispatchDueReminders(ctx context.Context) {
	rows, err := s.db.QueryContext(ctx, `SELECT r.id, r.food_id, u.openid, f.name, f.quantity, f.expiry_date
        FROM reminder_jobs r JOIN foods f ON f.id=r.food_id JOIN users u ON u.id=r.user_id
        WHERE r.status='pending' AND julianday(r.remind_at)<=julianday('now') AND f.status='active'`)
	if err != nil {
		log.Printf("find reminder jobs: %v", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var job reminderDelivery
		if err := rows.Scan(&job.ID, &job.FoodID, &job.OpenID, &job.Name, &job.Quantity, &job.Expiry); err != nil {
			log.Printf("scan reminder: %v", err)
			continue
		}
		if err := s.sendReminder(ctx, job); err != nil {
			log.Printf("send reminder id=%s: %v", job.ID, err)
			if _, updateErr := s.db.ExecContext(ctx, "UPDATE reminder_jobs SET status='failed', error_message=? WHERE id=? AND status='pending'", err.Error(), job.ID); updateErr != nil {
				log.Printf("mark reminder failed: %v", updateErr)
			}
			continue
		}
		if _, err := s.db.ExecContext(ctx, "UPDATE reminder_jobs SET status='sent', sent_at=CURRENT_TIMESTAMP, error_message='' WHERE id=? AND status='pending'", job.ID); err != nil {
			log.Printf("mark reminder sent: %v", err)
		}
	}
}

func (s *Service) sendReminder(ctx context.Context, job reminderDelivery) error {
	if s.cfg.WeChatTemplateID == "" {
		return fmt.Errorf("WECHAT_TEMPLATE_ID is not configured")
	}
	token, err := s.wechatAccessToken(ctx)
	if err != nil {
		return err
	}
	body, err := json.Marshal(reminderPayload(job, s.cfg.WeChatTemplateID))
	if err != nil {
		return fmt.Errorf("marshal reminder payload: %w", err)
	}
	endpoint := "https://api.weixin.qq.com/cgi-bin/message/subscribe/send?access_token=" + url.QueryEscape(token)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create reminder request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := s.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send reminder request: %w", err)
	}
	defer response.Body.Close()
	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode reminder response: %w", err)
	}
	if response.StatusCode != http.StatusOK || result.ErrCode != 0 {
		return fmt.Errorf("WeChat reminder rejected: %d %s", result.ErrCode, result.ErrMsg)
	}
	return nil
}

func reminderPayload(job reminderDelivery, templateID string) map[string]any {
	today := time.Now().In(time.Local)
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)
	expiry := time.Date(job.Expiry.Year(), job.Expiry.Month(), job.Expiry.Day(), 0, 0, 0, 0, time.Local)
	daysLeft := int(expiry.Sub(today).Hours() / 24)
	if daysLeft < 0 {
		daysLeft = 0
	}
	return map[string]any{
		"touser":      job.OpenID,
		"template_id": templateID,
		"page":        "pages/detail/index?id=" + job.FoodID,
		"data": map[string]any{
			"date1":   map[string]string{"value": expiry.Format("2006年01月02日")},
			"number2": map[string]string{"value": strconv.Itoa(daysLeft)},
			"thing6":  map[string]string{"value": job.Name},
			"number7": map[string]string{"value": strconv.FormatFloat(job.Quantity, 'f', -1, 64)},
		},
	}
}

func (s *Service) wechatAccessToken(ctx context.Context) (string, error) {
	s.accessTokenMu.Lock()
	defer s.accessTokenMu.Unlock()
	if s.accessToken != "" && time.Now().Before(s.accessTokenExpires) {
		return s.accessToken, nil
	}
	if s.cfg.WeChatAppID == "" || s.cfg.WeChatAppSecret == "" {
		return "", fmt.Errorf("WeChat credentials are not configured")
	}
	endpoint := "https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=" + url.QueryEscape(s.cfg.WeChatAppID) + "&secret=" + url.QueryEscape(s.cfg.WeChatAppSecret)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("create access token request: %w", err)
	}
	response, err := s.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("request access token: %w", err)
	}
	defer response.Body.Close()
	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode access token response: %w", err)
	}
	if response.StatusCode != http.StatusOK || result.ErrCode != 0 || result.AccessToken == "" {
		return "", fmt.Errorf("WeChat access token rejected: %d %s", result.ErrCode, result.ErrMsg)
	}
	s.accessToken = result.AccessToken
	s.accessTokenExpires = time.Now().Add(time.Duration(result.ExpiresIn-300) * time.Second)
	return s.accessToken, nil
}
