package app

import (
	"testing"
	"time"
)

func TestReminderPayloadUsesConfiguredTemplateFields(t *testing.T) {
	expiry := time.Date(2030, time.January, 2, 0, 0, 0, 0, time.Local)
	payload := reminderPayload(reminderDelivery{FoodID: "food-1", OpenID: "openid-1", Name: "鲜牛奶", Quantity: 2, Expiry: expiry}, "template-1")
	if payload["touser"] != "openid-1" || payload["template_id"] != "template-1" || payload["page"] != "pages/detail/index?id=food-1" {
		t.Fatalf("unexpected envelope: %#v", payload)
	}
	data := payload["data"].(map[string]any)
	if got := data["date1"].(map[string]string)["value"]; got != "2030年01月02日" {
		t.Fatalf("unexpected date1: %s", got)
	}
	if got := data["thing6"].(map[string]string)["value"]; got != "鲜牛奶" {
		t.Fatalf("unexpected thing6: %s", got)
	}
	if got := data["number7"].(map[string]string)["value"]; got != "2" {
		t.Fatalf("unexpected number7: %s", got)
	}
}
