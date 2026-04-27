<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import CsvViewer from '$lib/components/CsvViewer.svelte';
	import { fetchRunDetail, fetchAnalysis } from '$lib/stores/history';
	import type { ReportRow } from '$lib/types';

	let runId = $derived($page.params.runId);
	let detail = $state<Record<string, string> | null>(null);
	let analysis = $state<ReportRow[]>([]);
	let loading = $state(true);

	onMount(async () => {
		const [d, a] = await Promise.all([
			fetchRunDetail(runId),
			fetchAnalysis(runId)
		]);
		detail = d;
		analysis = a;
		loading = false;
	});
</script>

<div class="run-detail-page">
	<a href="/history" class="back-link">&#8592; Back to History</a>

	<div class="card">
		<h2>Run: {runId}</h2>
		{#if loading}
			<p class="text-muted">Loading...</p>
		{:else if detail}
			<table class="detail-table">
				<tbody>
					{#each Object.entries(detail) as [key, value]}
						<tr>
							<td class="label">{key}</td>
							<td>{value}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{:else}
			<p class="text-muted">No metadata available for this run.</p>
		{/if}
	</div>

	<div class="card">
		<h3>Analysis Report</h3>
		{#if loading}
			<p class="text-muted">Loading analysis...</p>
		{:else}
			<CsvViewer rows={analysis} />
		{/if}
	</div>
</div>

<style>
	.run-detail-page { display: flex; flex-direction: column; gap: 1rem; }
	.back-link {
		color: var(--color-primary);
		text-decoration: none;
		font-size: 0.9rem;
	}
	.back-link:hover { text-decoration: underline; }
	h2 { font-size: 1.25rem; margin-bottom: 0.75rem; }
	h3 { font-size: 1rem; margin-bottom: 0.75rem; }
	.detail-table { width: 100%; border-collapse: collapse; }
	.detail-table td {
		padding: 0.375rem 0.75rem;
		border-bottom: 1px solid var(--color-border);
		font-size: 0.9rem;
	}
	.label {
		font-weight: 500;
		color: var(--color-text-muted);
		white-space: nowrap;
		width: 1%;
	}
</style>
