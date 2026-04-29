<script lang="ts">
	import type { ReportRow } from '$lib/types';

	let { rows }: { rows: ReportRow[] } = $props();
	let sortCol = $state<keyof ReportRow>('entity_type');
	let sortAsc = $state(true);

	let summary = $derived(() => {
		const total = rows.length;
		const success = rows.filter((r) => r.outcome === 'success').length;
		const failure = total - success;
		return { total, success, failure };
	});

	let sorted = $derived(() => {
		return [...rows].sort((a, b) => {
			const va = a[sortCol] || '';
			const vb = b[sortCol] || '';
			const cmp = va.localeCompare(vb);
			return sortAsc ? cmp : -cmp;
		});
	});

	function toggleSort(col: keyof ReportRow) {
		if (sortCol === col) {
			sortAsc = !sortAsc;
		} else {
			sortCol = col;
			sortAsc = true;
		}
	}

	const columns: { key: keyof ReportRow; label: string }[] = [
		{ key: 'entity_type', label: 'Type' },
		{ key: 'entity_name', label: 'Name' },
		{ key: 'organization', label: 'Organization' },
		{ key: 'http_status', label: 'Status' },
		{ key: 'outcome', label: 'Outcome' },
		{ key: 'error_message', label: 'Error' }
	];
</script>

{#if rows.length === 0}
	<p class="text-muted">No analysis data available.</p>
{:else}
	<div id="csv-summary-bar" class="summary-bar">
		<span>Total: <strong>{summary().total}</strong></span>
		<span class="text-success">Success: <strong>{summary().success}</strong></span>
		<span class="text-error">Failure: <strong>{summary().failure}</strong></span>
	</div>

	<div id="csv-table-wrap" class="table-wrap">
		<table>
			<thead>
				<tr>
					{#each columns as col}
						<th onclick={() => toggleSort(col.key)} class="sortable">
							{col.label}
							{#if sortCol === col.key}
								<span class="sort-arrow">{sortAsc ? '\u25B2' : '\u25BC'}</span>
							{/if}
						</th>
					{/each}
				</tr>
			</thead>
			<tbody>
				{#each sorted() as row}
					<tr>
						<td>{row.entity_type}</td>
						<td class="mono">{row.entity_name}</td>
						<td>{row.organization}</td>
						<td class="mono">{row.http_status}</td>
						<td>
							<span class="outcome" class:success={row.outcome === 'success'} class:failure={row.outcome !== 'success'}>
								{row.outcome}
							</span>
						</td>
						<td class="error-col">{row.error_message || ''}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}

<style>
	.summary-bar {
		display: flex;
		gap: 1.5rem;
		padding: 0.625rem 0;
		font-size: 0.9rem;
	}
	.table-wrap {
		overflow-x: auto;
	}
	table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.85rem;
	}
	th {
		text-align: left;
		padding: 0.5rem 0.625rem;
		border-bottom: 2px solid var(--color-border);
		font-size: 0.75rem;
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}
	th.sortable { cursor: pointer; user-select: none; }
	th.sortable:hover { color: var(--color-primary); }
	.sort-arrow { font-size: 0.6rem; margin-left: 0.25rem; }
	td {
		padding: 0.375rem 0.625rem;
		border-bottom: 1px solid var(--color-border);
	}
	tr:hover td { background: var(--color-bg); }
	.mono { font-family: 'SF Mono', Consolas, monospace; font-size: 0.8rem; }
	.outcome {
		display: inline-block;
		padding: 0.0625rem 0.375rem;
		border-radius: 3px;
		font-size: 0.75rem;
		font-weight: 500;
	}
	.outcome.success { background: var(--color-badge-success-bg); color: var(--color-success); }
	.outcome.failure { background: var(--color-badge-error-bg); color: var(--color-error); }
	.error-col {
		max-width: 200px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		color: var(--color-error);
		font-size: 0.8rem;
	}
</style>
