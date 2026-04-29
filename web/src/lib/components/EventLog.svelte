<script lang="ts">
	import { eventLog } from '$lib/stores/wizard';

	let logEl: HTMLDivElement;
	let entries = $derived($eventLog);

	$effect(() => {
		// Auto-scroll to bottom on new entries.
		if (entries.length > 0 && logEl) {
			logEl.scrollTop = logEl.scrollHeight;
		}
	});

	function msgClass(type: string): string {
		if (type.includes('error')) return 'text-error';
		if (type.includes('warning')) return 'text-warning';
		if (type.includes('success') || type === 'display_wizard_complete') return 'text-success';
		if (type === 'display_phase_progress') return 'text-phase';
		return '';
	}
</script>

<div id="event-log" class="log" bind:this={logEl}>
	{#each entries as entry}
		<div class="entry {msgClass(entry.type)}">
			<span class="time">{entry.timestamp.toLocaleTimeString()}</span>
			<span class="msg">{entry.message}</span>
		</div>
	{/each}
	{#if entries.length === 0}
		<p class="text-muted">No events yet. Start the wizard to begin.</p>
	{/if}
</div>

<style>
	.log {
		flex: 1;
		min-height: 100px;
		overflow-y: auto;
		font-family: 'SF Mono', 'Consolas', 'Monaco', monospace;
		font-size: 0.8rem;
		padding: 0.75rem;
		background: var(--color-bg);
		border: 1px solid var(--color-border);
		border-radius: var(--radius);
	}
	.entry {
		display: flex;
		gap: 0.75rem;
		padding: 0.125rem 0;
		line-height: 1.4;
	}
	.time {
		color: var(--color-text-muted);
		flex-shrink: 0;
	}
	.text-phase { color: var(--color-primary); font-weight: 600; }
</style>
