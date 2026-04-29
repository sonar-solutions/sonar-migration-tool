<script lang="ts">
	import type { RunInfo } from '$lib/types';

	let { runs }: { runs: RunInfo[] } = $props();
</script>

{#if runs.length === 0}
	<p class="text-muted">No migration runs found.</p>
{:else}
	<table class="run-table">
		<thead>
			<tr>
				<th>Run ID</th>
				<th>Source URL</th>
				<th>Analysis</th>
				<th>Report</th>
				<th></th>
			</tr>
		</thead>
		<tbody>
			{#each runs as run}
				<tr>
					<td class="mono">{run.run_id}</td>
					<td>{run.source_url || '—'}</td>
					<td>
						{#if run.has_analysis}
							<span class="badge badge-yes">Available</span>
						{:else}
							<span class="badge badge-no">None</span>
						{/if}
					</td>
					<td>
						{#if run.has_report}
							<span class="badge badge-yes">Available</span>
						{:else}
							<span class="badge badge-no">None</span>
						{/if}
					</td>
					<td>
						<a href="/history/{run.run_id}" class="btn-link">View</a>
					</td>
				</tr>
			{/each}
		</tbody>
	</table>
{/if}

<style>
	.run-table {
		width: 100%;
		border-collapse: collapse;
	}
	th {
		text-align: left;
		padding: 0.5rem 0.75rem;
		border-bottom: 2px solid var(--color-border);
		font-size: 0.8rem;
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}
	td {
		padding: 0.625rem 0.75rem;
		border-bottom: 1px solid var(--color-border);
		font-size: 0.9rem;
	}
	tr:hover td { background: var(--color-bg); }
	.mono { font-family: 'SF Mono', Consolas, monospace; }
	.badge {
		display: inline-block;
		padding: 0.125rem 0.5rem;
		border-radius: 999px;
		font-size: 0.75rem;
		font-weight: 500;
	}
	.badge-yes { background: var(--color-badge-success-bg); color: var(--color-success); }
	.badge-no { background: var(--color-badge-muted-bg); color: var(--color-text-muted); }
	.btn-link {
		color: var(--color-primary);
		text-decoration: none;
		font-size: 0.85rem;
		font-weight: 500;
	}
	.btn-link:hover { text-decoration: underline; }
</style>
