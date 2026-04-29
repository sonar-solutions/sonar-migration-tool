<script lang="ts">
	import { respondToPrompt } from '$lib/stores/wizard';
	import type { ServerMessage } from '$lib/types';

	const CLOUD_OPTIONS = [
		{ label: 'SonarQube Cloud (sonarcloud.io)', value: 'https://sonarcloud.io' },
		{ label: 'SonarQube Cloud US (sonarqube.us)', value: 'https://sonarqube.us' },
		{ label: 'Other', value: '' }
	];

	let { prompt }: { prompt: ServerMessage } = $props();
	let selected = $state(CLOUD_OPTIONS[0].value);
	let customURL = $state('');
	let error = $state('');
	let isOther = $derived(selected === '');
	let inputEl: HTMLInputElement;

	$effect(() => {
		if (isOther && inputEl) {
			inputEl.focus();
		}
	});

	function submit() {
		const value = isOther ? customURL : selected;
		if (!value) {
			error = 'Please enter a URL';
			return;
		}
		try {
			const url = new URL(value);
			if (url.protocol !== 'http:' && url.protocol !== 'https:') {
				error = 'URL must use http or https';
				return;
			}
		} catch {
			error = 'Invalid URL format';
			return;
		}
		error = '';
		respondToPrompt(value);
	}

	function onKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			submit();
		}
	}
</script>

<div id="prompt-cloud-url" class="prompt">
	<label for="cloud-url-select">{prompt.message}</label>
	<div id="prompt-cloud-url-row" class="input-row">
		<select id="cloud-url-select" bind:value={selected}>
			{#each CLOUD_OPTIONS as opt}
				<option value={opt.value}>{opt.label}</option>
			{/each}
		</select>
		<button class="btn-primary" onclick={submit} disabled={isOther && !customURL}>Submit</button>
	</div>
	{#if isOther}
		<div id="prompt-cloud-url-custom" class="input-row">
			<input
				id="cloud-url-custom-input"
				type="url"
				bind:value={customURL}
				bind:this={inputEl}
				onkeydown={onKeydown}
				placeholder="https://your-sonarqube-cloud.example.com/"
			/>
		</div>
	{/if}
	{#if error}
		<p class="text-error">{error}</p>
	{/if}
</div>

<style>
	.prompt { display: flex; flex-direction: column; gap: 0.5rem; }
	label { font-weight: 500; }
	.input-row { display: flex; gap: 0.5rem; }
	.input-row input { flex: 1; }
	select {
		flex: 1;
		padding: 0.625rem 0.75rem;
		border: 1px solid var(--color-border);
		border-radius: var(--radius);
		font-size: 0.9rem;
		background: var(--color-surface);
		color: var(--color-text);
		outline: none;
		cursor: pointer;
	}
	select:focus {
		border-color: var(--color-primary);
		box-shadow: 0 0 0 3px rgba(74, 108, 247, 0.15);
	}
</style>
