<script lang="ts">
	import WizardStepper from '$lib/components/WizardStepper.svelte';
	import PromptURL from '$lib/components/PromptURL.svelte';
	import PromptText from '$lib/components/PromptText.svelte';
	import PromptPassword from '$lib/components/PromptPassword.svelte';
	import PromptConfirm from '$lib/components/PromptConfirm.svelte';
	import PromptConfirmReview from '$lib/components/PromptConfirmReview.svelte';
	import EventLog from '$lib/components/EventLog.svelte';
	import { wizardStatus, pendingPrompt, startWizard, cancelWizard } from '$lib/stores/wizard';
	import { connectionState } from '$lib/stores/websocket';

	let status = $derived($wizardStatus);
	let prompt = $derived($pendingPrompt);
	let connected = $derived($connectionState === 'open');
</script>

<div class="wizard-page">
	<div class="card">
		<WizardStepper />

		<div class="controls">
			{#if status === 'idle' || status === 'finished' || status === 'error'}
				<button class="btn-primary" onclick={startWizard} disabled={!connected}>
					{status === 'idle' ? 'Start Wizard' : 'Start New Wizard'}
				</button>
			{:else if status === 'running'}
				<button class="btn-danger" onclick={cancelWizard}>Cancel</button>
			{/if}

			{#if status === 'finished'}
				<p class="text-success status-msg">Migration complete!</p>
			{:else if status === 'error'}
				<p class="text-error status-msg">Wizard encountered an error. Check the log below.</p>
			{/if}
		</div>
	</div>

	{#if prompt}
		<div class="card prompt-card">
			{#if prompt.type === 'prompt_url'}
				<PromptURL {prompt} />
			{:else if prompt.type === 'prompt_text'}
				<PromptText {prompt} />
			{:else if prompt.type === 'prompt_password'}
				<PromptPassword {prompt} />
			{:else if prompt.type === 'prompt_confirm'}
				<PromptConfirm {prompt} />
			{:else if prompt.type === 'prompt_confirm_review'}
				<PromptConfirmReview {prompt} />
			{/if}
		</div>
	{/if}

	<div class="card">
		<h3>Event Log</h3>
		<EventLog />
	</div>
</div>

<style>
	.wizard-page {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}
	.controls {
		display: flex;
		align-items: center;
		gap: 1rem;
		padding-top: 0.5rem;
	}
	.status-msg { margin: 0; }
	.prompt-card {
		border-left: 3px solid var(--color-primary);
	}
	h3 {
		font-size: 0.9rem;
		margin-bottom: 0.75rem;
		color: var(--color-text-muted);
	}
</style>
