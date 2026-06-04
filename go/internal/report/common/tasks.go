// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package common

import (
	"time"

	"github.com/sonar-solutions/sonar-migration-tool/internal/report"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

// ProcessTasks extracts background task statistics from getTasks and getProjectTasks.
func ProcessTasks(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping) map[string]map[string]map[string]any {
	tasks := make(map[string]map[string]map[string]any)
	for _, key := range []string{"getTasks", "getProjectTasks"} {
		for _, item := range readData(dir, mapping, key) {
			sid := serverID(idMap, item.ServerURL)
			taskType := report.ExtractString(item.Data, "$.type")
			if taskType == "" {
				continue
			}
			ensureTaskEntry(tasks, sid, taskType)
			processTask(tasks[sid][taskType], item.Data, sid, taskType)
		}
	}
	return tasks
}

func ensureTaskEntry(tasks map[string]map[string]map[string]any, sid, taskType string) {
	if tasks[sid] == nil {
		tasks[sid] = make(map[string]map[string]any)
	}
	if tasks[sid][taskType] == nil {
		tasks[sid][taskType] = makeTaskTemplate(sid, taskType)
	}
}

func makeTaskTemplate(sid, taskType string) map[string]any {
	return map[string]any{
		"server_id": sid, "type": taskType,
		"total": 0, "succeeded": 0, "failed": 0, "canceled": 0,
		"min_queue_time": float64(0), "max_queue_time": float64(0),
		"total_queue_time": float64(0), "avg_queue_time": float64(0),
		"min_execution_time": float64(0), "max_execution_time": float64(0),
		"total_execution_time": float64(0), "avg_execution_time": float64(0),
		"first_run": nil, "last_run": nil,
	}
}

func processTask(entry map[string]any, data map[string]any, sid, taskType string) {
	submitted := report.ExtractString(data, "$.submittedAt")
	started := report.ExtractString(data, "$.startedAt")
	submittedTime, ok1 := parseSQDate(submitted)
	startedTime, ok2 := parseSQDate(started)
	if !ok1 || !ok2 {
		return
	}

	execTimeMs := report.ExtractFloat(data, "$.executionTimeMs", 0)
	queueTimeMs := startedTime.Sub(submittedTime).Seconds() * 1000

	total := entry["total"].(int) + 1
	entry["total"] = total
	entry["server_id"] = sid
	entry["type"] = taskType

	updateTaskStatusCounts(entry, data)
	updateTaskTiming(entry, queueTimeMs, execTimeMs, startedTime, total)
}

func updateTaskStatusCounts(entry, data map[string]any) {
	status := report.ExtractString(data, "$.status")
	switch status {
	case "SUCCESS":
		entry["succeeded"] = entry["succeeded"].(int) + 1
	case "FAILED":
		entry["failed"] = entry["failed"].(int) + 1
	case "CANCELED":
		entry["canceled"] = entry["canceled"].(int) + 1
	}
}

func updateTaskTiming(entry map[string]any, queueMs, execMs float64, startedTime time.Time, total int) {
	if total == 1 {
		entry["min_queue_time"] = queueMs
		entry["max_queue_time"] = queueMs
		entry["min_execution_time"] = execMs
		entry["max_execution_time"] = execMs
		entry["first_run"] = startedTime
		entry["last_run"] = startedTime
	} else {
		entry["min_queue_time"] = minFloat(entry["min_queue_time"].(float64), queueMs)
		entry["max_queue_time"] = maxFloat(entry["max_queue_time"].(float64), queueMs)
		entry["min_execution_time"] = minFloat(entry["min_execution_time"].(float64), execMs)
		entry["max_execution_time"] = maxFloat(entry["max_execution_time"].(float64), execMs)
		updateTimeRange(entry, startedTime)
	}
	totalQueue := entry["total_queue_time"].(float64) + queueMs
	totalExec := entry["total_execution_time"].(float64) + execMs
	entry["total_queue_time"] = totalQueue
	entry["total_execution_time"] = totalExec
	entry["avg_queue_time"] = totalQueue / float64(total)
	entry["avg_execution_time"] = totalExec / float64(total)
}

func updateTimeRange(entry map[string]any, t time.Time) {
	if first, ok := entry["first_run"].(time.Time); ok && t.Before(first) {
		entry["first_run"] = t
	}
	if last, ok := entry["last_run"].(time.Time); ok && t.After(last) {
		entry["last_run"] = t
	}
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// GenerateTaskMarkdown generates the Tasks markdown section.
func GenerateTaskMarkdown(dir string, mapping structure.ExtractMapping, idMap ServerIDMapping) string {
	tasks := ProcessTasks(dir, mapping, idMap)
	var rows []map[string]any
	for _, serverTasks := range tasks {
		for _, task := range serverTasks {
			rows = append(rows, task)
		}
	}
	return report.GenerateSection(
		[]report.Column{
			{Header: "Server ID", Key: "server_id"}, {Header: "Type", Key: "type"},
			{Header: "Total", Key: "total"}, {Header: "Succeeded", Key: "succeeded"}, {Header: "Failed", Key: "failed"}, {Header: "Canceled", Key: "canceled"},
			{Header: "Min Queue Time (ms)", Key: "min_queue_time"}, {Header: "Max Queue Time (ms)", Key: "max_queue_time"},
			{Header: "Avg Queue Time (ms)", Key: "avg_queue_time"},
			{Header: "Min Execution Time (ms)", Key: "min_execution_time"}, {Header: "Max Execution Time (ms)", Key: "max_execution_time"},
			{Header: "Avg Execution Time (ms)", Key: "avg_execution_time"},
			{Header: "First Run", Key: "first_run"}, {Header: "Last Run", Key: "last_run"},
		},
		rows,
		report.WithTitle("Tasks (Past 30 Days)", 3),
	)
}
