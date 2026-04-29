import type { RunInfo, ReportRow } from '$lib/types';

export async function fetchRuns(): Promise<RunInfo[]> {
	const resp = await fetch('/api/runs');
	if (!resp.ok) return [];
	return resp.json();
}

export async function fetchRunDetail(runId: string): Promise<Record<string, string> | null> {
	const resp = await fetch(`/api/runs/${runId}`);
	if (!resp.ok) return null;
	return resp.json();
}

export async function fetchAnalysis(runId: string): Promise<ReportRow[]> {
	const resp = await fetch(`/api/runs/${runId}/analysis`);
	if (!resp.ok) return [];
	return resp.json();
}

export async function fetchReport(type: 'migration' | 'maturity'): Promise<string> {
	const resp = await fetch(`/api/reports/${type}`);
	if (!resp.ok) return '';
	return resp.text();
}
