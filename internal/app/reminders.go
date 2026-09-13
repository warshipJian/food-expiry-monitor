package app

import (
	"context"
	"log"
	"time"
)

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
	rows, err := s.db.QueryContext(ctx, `SELECT r.id, f.name, f.expiry_date FROM reminder_jobs r JOIN foods f ON f.id=r.food_id WHERE r.status='pending' AND r.remind_at<=CURRENT_TIMESTAMP AND f.status='active'`)
	if err != nil {
		log.Printf("find reminder jobs: %v", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, name string
		var expiry time.Time
		if err := rows.Scan(&id, &name, &expiry); err != nil {
			log.Printf("scan reminder: %v", err)
			continue
		}
		// Wire the WeChat subscribeMessage.send call here. Until credentials and
		// a reviewed template are configured, do not pretend a message was sent.
		log.Printf("reminder due id=%s food=%q expiry=%s", id, name, expiry.Format(dateLayout))
		if _, err := s.db.ExecContext(ctx, "UPDATE reminder_jobs SET status='failed', error_message=? WHERE id=? AND status='pending'", "WeChat sender is not configured", id); err != nil {
			log.Printf("mark reminder failed: %v", err)
		}
	}
}
