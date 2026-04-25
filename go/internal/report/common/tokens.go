package common

import (
	"strings"

	"github.com/sonar-solutions/sonar-migration-tool/internal/report"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

// ProcessTokens extracts user token statistics from getUserTokens data.
func ProcessTokens(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping) map[string]map[string]any {
	tokens := make(map[string]map[string]any)
	for _, item := range readData(dir, mapping, "getUserTokens") {
		sid := serverID(idMap, item.ServerURL)
		ensureTokenEntry(tokens, sid)
		tokenName := report.ExtractString(item.Data, "$.name")
		tokenType := report.ExtractString(item.Data, "$.type")
		if tokenName == "" || tokenType == "" {
			continue
		}
		if strings.Contains(strings.ToLower(tokenName), "sonarlint") || strings.ToUpper(tokenType) != "USER_TOKEN" {
			continue
		}
		updateTokenStats(tokens[sid], item.Data, sid)
	}
	return tokens
}

func ensureTokenEntry(tokens map[string]map[string]any, sid string) {
	if tokens[sid] == nil {
		tokens[sid] = map[string]any{
			"server_id":     sid,
			"total_tokens":  0,
			"expired_tokens": 0,
			"active_tokens": 0,
			"recent_tokens": 0,
			"users":         make(map[string]bool),
			"user_count":    0,
		}
	}
}

func updateTokenStats(entry map[string]any, data map[string]any, sid string) {
	entry["server_id"] = sid
	entry["total_tokens"] = entry["total_tokens"].(int) + 1

	if report.ExtractBool(data, "$.isExpired") {
		entry["expired_tokens"] = entry["expired_tokens"].(int) + 1
	}
	entry["active_tokens"] = entry["total_tokens"].(int) - entry["expired_tokens"].(int)

	lastConn := report.ExtractString(data, "$.lastConnectionDate")
	if t, ok := parseSQDate(lastConn); ok && isRecent(t) {
		entry["recent_tokens"] = entry["recent_tokens"].(int) + 1
	}

	login := report.ExtractString(data, "$.login")
	if login != "" {
		users := entry["users"].(map[string]bool)
		users[login] = true
		entry["user_count"] = len(users)
	}
}

// GenerateTokenMarkdown generates the Tokens markdown section.
func GenerateTokenMarkdown(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping) string {
	tokens := ProcessTokens(dir, mapping, idMap)
	var rows []map[string]any
	for _, entry := range tokens {
		rows = append(rows, entry)
	}
	return report.GenerateSection(
		[]report.Column{
			{"Server ID", "server_id"}, {"Total Tokens", "total_tokens"}, {"Active Tokens", "active_tokens"},
			{"Recently Used Tokens", "recent_tokens"}, {"Users", "user_count"},
		},
		rows,
		report.WithTitle("Tokens", 3),
	)
}
