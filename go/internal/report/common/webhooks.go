package common

import (
	"github.com/sonar-solutions/sonar-migration-tool/internal/report"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

// ProcessWebhooks extracts webhook details and delivery statistics.
func ProcessWebhooks(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping) map[string]map[string]map[string]any {
	webhooks := make(map[string]map[string]map[string]any)

	for _, key := range []string{"getWebhooks", "getProjectWebhooks"} {
		for _, item := range readData(dir, mapping, key) {
			sid := serverID(idMap, item.ServerURL)
			processWebhookEntry(webhooks, sid, item.Data)
		}
	}

	for _, key := range []string{"getWebhookDeliveries", "getProjectWebhookDeliveries"} {
		for _, item := range readData(dir, mapping, key) {
			sid := serverID(idMap, item.ServerURL)
			processDeliveryEntry(webhooks, sid, item.Data)
		}
	}
	return webhooks
}

func processWebhookEntry(webhooks map[string]map[string]map[string]any, sid string, data map[string]any) {
	name := report.ExtractString(data, "$.name")
	if name == "" {
		return
	}
	if webhooks[sid] == nil {
		webhooks[sid] = make(map[string]map[string]any)
	}
	webhooks[sid][name] = map[string]any{
		"server_id":    sid,
		"name":         name,
		"url":          report.ExtractString(data, "$.url"),
		"project":      report.ExtractString(data, "$.projectKey"),
		"has_secret":   report.ExtractBool(data, "$.hasSecret"),
		"deliveries":   0,
		"successes":    0,
		"failures":     0,
		"last_success": nil,
		"last_error":   nil,
	}
}

func processDeliveryEntry(webhooks map[string]map[string]map[string]any, sid string, data map[string]any) {
	name := report.ExtractString(data, "$.name")
	if webhooks[sid] == nil || webhooks[sid][name] == nil {
		return
	}
	wh := webhooks[sid][name]
	wh["deliveries"] = wh["deliveries"].(int) + 1

	deliveryDate := report.ExtractString(data, "$.at")
	t, ok := parseSQDate(deliveryDate)
	if !ok {
		return
	}

	if report.ExtractBool(data, "$.success") {
		wh["successes"] = wh["successes"].(int) + 1
		wh["last_success"] = t
	} else {
		wh["failures"] = wh["failures"].(int) + 1
		wh["last_error"] = t
	}
}

// GenerateWebhookMarkdown generates the Webhooks markdown section.
func GenerateWebhookMarkdown(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping) string {
	webhooks := ProcessWebhooks(dir, mapping, idMap)
	var rows []map[string]any
	for _, serverWebhooks := range webhooks {
		for _, wh := range serverWebhooks {
			rows = append(rows, wh)
		}
	}
	return report.GenerateSection(
		[]report.Column{
			{"Server ID", "server_id"}, {"Webhook Name", "name"}, {"URL", "url"},
			{"Project", "project"}, {"Deliveries", "deliveries"},
			{"Successful Deliveries", "successes"}, {"Failed Deliveries", "failures"},
			{"Last Successful Delivery", "last_success"}, {"Last Failed Delivery", "last_error"},
		},
		rows,
		report.WithTitle("Webhooks", 3),
	)
}
