<script lang="ts">
	import { respondToPrompt } from '$lib/stores/wizard';
	import type { ServerMessage } from '$lib/types';

	let { prompt }: { prompt: ServerMessage } = $props();
	let value = $state('');
	let error = $state('');
	let inputEl: HTMLInputElement;

	$effect(() => {
		inputEl?.focus();
	});

	function submit() {
		if (prompt.validate) {
			try {
				const url = new URL(value);
				if (url.protocol !== 'http:' && url.protocol !== 'https:') {
					error = 'URL must use http or https';
					return;
				}
				if (!url.hostname) {
					error = 'URL must have a hostname';
					return;
				}
			} catch {
				error = 'Invalid URL format';
				return;
			}
		}
		error = '';
		respondToPrompt(value);
		value = '';
	}

	function onKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') submit();
	}
</script>

<div id="prompt-url" class="prompt">
	<label for="url-input">{prompt.message}</label>
	<div id="prompt-url-input-row" class="input-row">
		<input
			id="url-input"
			type="url"
			bind:value
			bind:this={inputEl}
			onkeydown={onKeydown}
			placeholder="https://sonarqube.example.com/"
		/>
		<button class="btn-primary" onclick={submit}>Submit</button>
	</div>
	{#if error}
		<p class="text-error">{error}</p>
	{/if}
</div>

<style>
	.prompt { display: flex; flex-direction: column; gap: 0.5rem; }
	label { font-weight: 500; }
	.input-row { display: flex; gap: 0.5rem; }
	.input-row input { flex: 1; }
</style>
