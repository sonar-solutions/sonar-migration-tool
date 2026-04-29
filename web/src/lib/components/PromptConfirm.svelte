<script lang="ts">
	import { respondToPrompt, latestSummary } from '$lib/stores/wizard';
	import type { ServerMessage } from '$lib/types';

	let { prompt }: { prompt: ServerMessage } = $props();
	let summary = $derived($latestSummary);
</script>

<div id="prompt-confirm" class="prompt">
	{#if summary}
		<h4 class="summary-title">{summary.title}</h4>
		<table id="prompt-confirm-summary" class="summary-table">
			<tbody>
				{#each summary.stats as stat}
					<tr>
						<td class="stat-key">{stat.key}</td>
						<td class="stat-value">{stat.value}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}
	<p class="message">{prompt.message}</p>
	<div id="prompt-confirm-buttons" class="button-row">
		<button class="btn-primary" onclick={() => respondToPrompt(true)}>Yes</button>
		<button class="btn-secondary" onclick={() => respondToPrompt(false)}>No</button>
	</div>
</div>

<style>
	.prompt { display: flex; flex-direction: column; gap: 0.75rem; }
	.message { font-weight: 500; }
	.button-row { display: flex; gap: 0.5rem; }
	.summary-title {
		font-size: 0.9rem;
		font-weight: 600;
		color: var(--color-text);
	}
	.summary-table {
		width: 100%;
		border-collapse: collapse;
	}
	.summary-table td {
		padding: 0.25rem 0.5rem;
		border-bottom: 1px solid var(--color-border);
		font-size: 0.85rem;
	}
	.stat-key {
		font-weight: 500;
		color: var(--color-text-muted);
		white-space: nowrap;
		width: 1%;
	}
	.stat-value {
		color: var(--color-text);
	}
</style>
