<script lang="ts">
	import WizardStepper from '$lib/components/WizardStepper.svelte';
	import PromptURL from '$lib/components/PromptURL.svelte';
	import PromptCloudURL from '$lib/components/PromptCloudURL.svelte';
	import PromptText from '$lib/components/PromptText.svelte';
	import PromptPassword from '$lib/components/PromptPassword.svelte';
	import PromptConfirm from '$lib/components/PromptConfirm.svelte';
	import PromptConfirmReview from '$lib/components/PromptConfirmReview.svelte';
	import EventLog from '$lib/components/EventLog.svelte';
	import { wizardStatus, pendingPrompt, startWizard, cancelWizard, isProcessing, latestDisplayMessage } from '$lib/stores/wizard';
	import { connectionState } from '$lib/stores/websocket';

	let status = $derived($wizardStatus);
	let prompt = $derived($pendingPrompt);
	let connected = $derived($connectionState === 'open');
	let processing = $derived($isProcessing);
	let processingMsg = $derived($latestDisplayMessage);
</script>

<div id="wizard-page" class="wizard-page">
	<div id="phases-card" class="card phases-card">
		<h3>Migration Phases</h3>
		<WizardStepper />
		<div id="wizard-controls" class="controls">
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
				<p class="text-error status-msg">Wizard encountered an error.</p>
			{/if}
		</div>
	</div>

	<div id="right-panel" class="right-panel">
		{#if prompt}
			<div id="prompt-card" class="card prompt-card">
				{#if prompt.type === 'prompt_url' && prompt.message?.includes('Cloud URL')}
					<PromptCloudURL {prompt} />
				{:else if prompt.type === 'prompt_url'}
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
		{:else if processing}
			<div id="processing-card" class="card processing-card">
				<div id="processing-spinner" class="spinner"></div>
				<span class="processing-text">{processingMsg || 'Working...'}</span>
			</div>
		{/if}

		<div id="event-log-card" class="card event-log-card">
			<h3>Event Log</h3>
			<EventLog />
		</div>
	</div>
</div>

<style>
	.wizard-page {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1rem;
		height: calc(100vh - 6rem);
	}
	.phases-card {
		overflow-y: auto;
		display: flex;
		flex-direction: column;
	}
	.controls {
		margin-top: auto;
		padding-top: 1rem;
		display: flex;
		align-items: center;
		gap: 0.75rem;
		border-top: 1px solid var(--color-border);
	}
	.status-msg { margin: 0; font-size: 0.85rem; }
	.right-panel {
		display: flex;
		flex-direction: column;
		gap: 1rem;
		min-height: 0;
	}
	.prompt-card {
		border-left: 3px solid var(--color-primary);
		flex-shrink: 0;
	}
	.processing-card {
		display: flex;
		align-items: center;
		gap: 1rem;
		border-left: 3px solid var(--color-primary);
		flex-shrink: 0;
	}
	.event-log-card {
		flex: 1;
		min-height: 0;
		display: flex;
		flex-direction: column;
	}
	.spinner {
		width: 1.5rem;
		height: 1.5rem;
		border: 3px solid var(--color-border);
		border-top-color: var(--color-primary);
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
		flex-shrink: 0;
	}
	@keyframes spin {
		to { transform: rotate(360deg); }
	}
	.processing-text {
		color: var(--color-text-muted);
		font-size: 0.9rem;
	}
	h3 {
		font-size: 0.9rem;
		margin-bottom: 0.75rem;
		color: var(--color-text-muted);
	}
</style>
