<script lang="ts">
	import { currentPhase } from '$lib/stores/wizard';
	import { PHASES } from '$lib/types';

	let phase = $derived($currentPhase);
</script>

<div class="stepper">
	{#each PHASES as p}
		{@const isActive = phase?.phase === p.key}
		{@const isComplete = phase ? p.index < phase.index : false}
		<div class="step" class:active={isActive} class:complete={isComplete}>
			<div class="step-circle">
				{#if isComplete}
					<span class="check">&#10003;</span>
				{:else}
					{p.index}
				{/if}
			</div>
			<span class="step-label">{p.name}</span>
		</div>
		{#if p.index < PHASES.length}
			<div class="step-connector" class:complete={isComplete}></div>
		{/if}
	{/each}
</div>

<style>
	.stepper {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0;
		padding: 1.5rem 0;
	}
	.step {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.375rem;
	}
	.step-circle {
		width: 2.25rem;
		height: 2.25rem;
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 0.85rem;
		font-weight: 600;
		background: var(--color-bg);
		border: 2px solid var(--color-border);
		color: var(--color-text-muted);
		transition: all 0.2s;
	}
	.step.active .step-circle {
		background: var(--color-primary);
		border-color: var(--color-primary);
		color: white;
	}
	.step.complete .step-circle {
		background: var(--color-success);
		border-color: var(--color-success);
		color: white;
	}
	.check { font-size: 1rem; }
	.step-label {
		font-size: 0.75rem;
		color: var(--color-text-muted);
		white-space: nowrap;
	}
	.step.active .step-label { color: var(--color-primary); font-weight: 600; }
	.step.complete .step-label { color: var(--color-success); }
	.step-connector {
		flex: 1;
		height: 2px;
		min-width: 2rem;
		max-width: 4rem;
		background: var(--color-border);
		margin-bottom: 1.5rem;
		transition: background 0.2s;
	}
	.step-connector.complete { background: var(--color-success); }
</style>
