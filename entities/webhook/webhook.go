package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/git-bug/git-bug/repository"
)

const webhookConfigPrefix = "git-bug.webhook."

type Webhook struct {
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Events []string `json:"events"`
}

type WebhookEvent struct {
	Event     string      `json:"event"`
	Timestamp time.Time   `json:"timestamp"`
	BugID     string      `json:"bug_id"`
	Data      interface{} `json:"data"`
}

func ListWebhooks(cfg repository.ConfigRead) ([]Webhook, error) {
	all, err := cfg.ReadAll(webhookConfigPrefix)
	if err != nil {
		return nil, err
	}

	webhookMap := make(map[string]*Webhook)
	for key, value := range all {
		if len(key) <= len(webhookConfigPrefix) {
			continue
		}
		rest := key[len(webhookConfigPrefix):]

		parts := splitConfigKey(rest)
		if len(parts) < 2 {
			continue
		}

		name := parts[0]
		field := parts[1]

		wh, ok := webhookMap[name]
		if !ok {
			wh = &Webhook{Name: name}
			webhookMap[name] = wh
		}

		switch field {
		case "url":
			wh.URL = value
		case "events":
			wh.Events = splitEvents(value)
		}
	}

	result := make([]Webhook, 0, len(webhookMap))
	for _, wh := range webhookMap {
		result = append(result, *wh)
	}
	return result, nil
}

func AddWebhook(cfg repository.ConfigWrite, name string, url string, events []string) error {
	prefix := webhookConfigPrefix + name + "."

	if err := cfg.StoreString(prefix+"url", url); err != nil {
		return err
	}
	if err := cfg.StoreString(prefix+"events", joinEvents(events)); err != nil {
		return err
	}

	return nil
}

func RemoveWebhook(cfg repository.ConfigWrite, name string) error {
	prefix := webhookConfigPrefix + name + "."
	return cfg.RemoveAll(prefix)
}

func TriggerWebhooks(cfg repository.ConfigRead, event string, bugID string, data interface{}) error {
	webhooks, err := ListWebhooks(cfg)
	if err != nil {
		return err
	}

	eventData := WebhookEvent{
		Event:     event,
		Timestamp: time.Now(),
		BugID:     bugID,
		Data:      data,
	}

	payload, err := json.Marshal(eventData)
	if err != nil {
		return err
	}

	for _, wh := range webhooks {
		if !matchesEvent(wh.Events, event) {
			continue
		}

		go sendWebhook(wh.URL, payload)
	}

	return nil
}

func sendWebhook(url string, payload []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	transport := &http.Transport{
		IdleConnTimeout:     5 * time.Second,
		TLSHandshakeTimeout: 3 * time.Second,
	}

	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: transport,
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "git-bug-webhook")

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

func matchesEvent(subscribed []string, event string) bool {
	if len(subscribed) == 0 {
		return true
	}
	for _, e := range subscribed {
		if e == event || e == "*" {
			return true
		}
	}
	return false
}

func splitConfigKey(s string) []string {
	var parts []string
	current := ""
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			if i+1 < len(s) && s[i+1] == '.' {
				current += "."
				i++
			} else {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(s[i])
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func splitEvents(s string) []string {
	if s == "" {
		return nil
	}
	var events []string
	current := ""
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			events = append(events, current)
			current = ""
		} else {
			current += string(s[i])
		}
	}
	if current != "" {
		events = append(events, current)
	}
	return events
}

func joinEvents(events []string) string {
	if len(events) == 0 {
		return ""
	}
	result := ""
	for i, e := range events {
		if i > 0 {
			result += ","
		}
		result += e
	}
	return result
}

func ValidEvents() []string {
	return []string{"create", "update", "close", "*"}
}

func ValidateEvent(event string) error {
	for _, e := range ValidEvents() {
		if e == event {
			return nil
		}
	}
	return fmt.Errorf("invalid event: %s. Valid events: %v", event, ValidEvents())
}
