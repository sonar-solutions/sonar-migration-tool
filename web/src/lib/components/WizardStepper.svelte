<script lang="ts">
	import { currentPhase, wizardStatus, phaseInputs } from '$lib/stores/wizard';
	import { PHASES } from '$lib/types';

	let phase = $derived($currentPhase);
	let status = $derived($wizardStatus);
	let inputs = $derived($phaseInputs);

	function phaseState(p: typeof PHASES[number]): 'active' | 'complete' | 'pending' {
		if (!phase || status === 'idle') return 'pending';
		if (phase.phase === p.key) return 'active';
		if (p.index < phase.index) return 'complete';
		if (status === 'finished' && p.index <= PHASES.length) return 'complete';
		return 'pending';
	}
</script>

<div id="wizard-stepper" class="stepper-vertical">
	{#each PHASES as p, i}
		{@const state = phaseState(p)}
		<div id="phase-{p.key}" class="phase-row" class:active={state === 'active'} class:complete={state === 'complete'}>
			<div id="phase-{p.key}-indicator" class="phase-indicator">
				<div id="phase-{p.key}-circle" class="phase-circle">
					{#if state === 'complete'}
						<span class="check">&#10003;</span>
					{:else}
						{p.index}
					{/if}
				</div>
				{#if i < PHASES.length - 1}
					<div id="phase-{p.key}-line" class="phase-line" class:complete={state === 'complete'}></div>
				{/if}
			</div>
			<div id="phase-{p.key}-content" class="phase-content">
				<div id="phase-{p.key}-header" class="phase-header">
					<span class="phase-name">{p.name}</span>
					<span class="phase-badge" class:badge-active={state === 'active'} class:badge-complete={state === 'complete'}>
						{state === 'active' ? 'In Progress' : state === 'complete' ? 'Complete' : 'Pending'}
					</span>
				</div>
				<p class="phase-desc">{p.description}</p>
				{#if inputs[p.key]?.length}
					<div id="phase-{p.key}-inputs" class="phase-inputs">
						{#each inputs[p.key] as input}
							<div class="phase-input-item">
								<span class="input-label">{input.label}</span>
								<span class="input-value">{input.value}</span>
							</div>
						{/each}
					</div>
				{/if}
			</div>
		</div>
	{/each}
</div>

<style>
	.stepper-vertical {
		display: flex;
		flex-direction: column;
	}
	.phase-row {
		display: flex;
		gap: 1rem;
		padding: 0.75rem 1rem;
		border-left: 3px solid transparent;
		border-radius: var(--radius);
		transition: background-color 0.15s, border-color 0.15s;
	}
	.phase-row.active {
		background: rgba(99, 137, 255, 0.08);
		border-left-color: var(--color-primary);
	}
	.phase-row.complete {
		opacity: 0.75;
	}
	.phase-indicator {
		display: flex;
		flex-direction: column;
		align-items: center;
		flex-shrink: 0;
	}
	.phase-circle {
		width: 2rem;
		height: 2rem;
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 0.8rem;
		font-weight: 600;
		background: var(--color-bg);
		border: 2px solid var(--color-border);
		color: var(--color-text-muted);
		transition: all 0.2s;
	}
	.phase-row.active .phase-circle {
		background: var(--color-primary);
		border-color: var(--color-primary);
		color: white;
	}
	.phase-row.complete .phase-circle {
		background: var(--color-success);
		border-color: var(--color-success);
		color: white;
	}
	.check { font-size: 0.9rem; }
	.phase-line {
		width: 2px;
		flex: 1;
		min-height: 1rem;
		background: var(--color-border);
		transition: background 0.2s;
	}
	.phase-line.complete {
		background: var(--color-success);
	}
	.phase-content {
		flex: 1;
		padding-bottom: 0.5rem;
	}
	.phase-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.5rem;
	}
	.phase-name {
		font-weight: 600;
		font-size: 0.9rem;
		color: var(--color-text);
	}
	.phase-row.active .phase-name {
		color: var(--color-primary);
	}
	.phase-badge {
		font-size: 0.7rem;
		padding: 0.125rem 0.5rem;
		border-radius: 999px;
		background: var(--color-badge-muted-bg);
		color: var(--color-text-muted);
		font-weight: 500;
	}
	.badge-active {
		background: rgba(99, 137, 255, 0.15);
		color: var(--color-primary);
	}
	.badge-complete {
		background: var(--color-badge-success-bg);
		color: var(--color-success);
	}
	.phase-desc {
		font-size: 0.8rem;
		color: var(--color-text-muted);
		margin-top: 0.25rem;
		line-height: 1.4;
	}
	.phase-inputs {
		margin-top: 0.375rem;
		display: flex;
		flex-direction: column;
		gap: 0.125rem;
	}
	.phase-input-item {
		display: flex;
		gap: 0.375rem;
		font-size: 0.75rem;
		line-height: 1.3;
	}
	.input-label {
		color: var(--color-text-muted);
		flex-shrink: 0;
	}
	.input-value {
		color: var(--color-text);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
</style>
