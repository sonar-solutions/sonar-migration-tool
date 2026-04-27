<script lang="ts">
	import { onMount } from 'svelte';
	import RunList from '$lib/components/RunList.svelte';
	import ReportViewer from '$lib/components/ReportViewer.svelte';
	import { fetchRuns, fetchReport } from '$lib/stores/history';
	import type { RunInfo } from '$lib/types';

	let runs = $state<RunInfo[]>([]);
	let loading = $state(true);
	let activeReport = $state<'migration' | 'maturity' | null>(null);
	let reportMarkdown = $state('');
	let reportLoading = $state(false);

	onMount(async () => {
		runs = await fetchRuns();
		loading = false;
	});

	async function loadReport(type: 'migration' | 'maturity') {
		if (activeReport === type) {
			activeReport = null;
			return;
		}
		reportLoading = true;
		reportMarkdown = await fetchReport(type);
		activeReport = type;
		reportLoading = false;
	}
</script>

<div class="history-page">
	<div class="card">
		<div class="header-row">
			<h2>Run History</h2>
			<div class="report-buttons">
				<button
					class="btn-secondary"
					class:active={activeReport === 'migration'}
					onclick={() => loadReport('migration')}
				>
					Migration Report
				</button>
				<button
					class="btn-secondary"
					class:active={activeReport === 'maturity'}
					onclick={() => loadReport('maturity')}
				>
					Maturity Report
				</button>
			</div>
		</div>

		{#if loading}
			<p class="text-muted">Loading runs...</p>
		{:else}
			<RunList {runs} />
		{/if}
	</div>

	{#if activeReport}
		<div class="card">
			<h3>{activeReport === 'migration' ? 'Migration' : 'Maturity'} Report</h3>
			{#if reportLoading}
				<p class="text-muted">Generating report...</p>
			{:else if reportMarkdown}
				<ReportViewer markdown={reportMarkdown} />
			{:else}
				<p class="text-muted">No data available to generate this report.</p>
			{/if}
		</div>
	{/if}
</div>

<style>
	.history-page { display: flex; flex-direction: column; gap: 1rem; }
	.header-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 1rem;
	}
	h2 { font-size: 1.25rem; }
	h3 { font-size: 1rem; margin-bottom: 0.75rem; }
	.report-buttons { display: flex; gap: 0.5rem; }
	.btn-secondary.active {
		background: var(--color-primary);
		color: white;
		border-color: var(--color-primary);
	}
</style>
