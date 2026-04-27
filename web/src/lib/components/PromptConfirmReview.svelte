<script lang="ts">
	import { respondToPrompt } from '$lib/stores/wizard';
	import type { ServerMessage } from '$lib/types';

	let { prompt }: { prompt: ServerMessage } = $props();
</script>

<div class="prompt">
	<h3>{prompt.title}</h3>
	<table class="review-table">
		<tbody>
			{#each prompt.details || [] as detail}
				<tr>
					<td class="label">{detail.key}</td>
					<td>{detail.value}</td>
				</tr>
			{/each}
		</tbody>
	</table>
	<p class="confirm-label">Are these values correct?</p>
	<div class="button-row">
		<button class="btn-primary" onclick={() => respondToPrompt(true)}>Yes, continue</button>
		<button class="btn-secondary" onclick={() => respondToPrompt(false)}>No, re-enter</button>
	</div>
</div>

<style>
	.prompt { display: flex; flex-direction: column; gap: 0.75rem; }
	h3 { font-size: 1rem; }
	.review-table {
		width: 100%;
		border-collapse: collapse;
	}
	.review-table td {
		padding: 0.375rem 0.75rem;
		border-bottom: 1px solid var(--color-border);
	}
	.label {
		font-weight: 500;
		white-space: nowrap;
		width: 1%;
		color: var(--color-text-muted);
	}
	.confirm-label { font-weight: 500; }
	.button-row { display: flex; gap: 0.5rem; }
</style>
