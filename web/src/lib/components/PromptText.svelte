<script lang="ts">
	import { respondToPrompt } from '$lib/stores/wizard';
	import type { ServerMessage } from '$lib/types';

	let { prompt }: { prompt: ServerMessage } = $props();
	let value = $state((prompt.default as string) || '');
	let inputEl: HTMLInputElement;

	$effect(() => {
		inputEl?.focus();
	});

	function submit() {
		respondToPrompt(value);
		value = '';
	}

	function onKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') submit();
	}
</script>

<div id="prompt-text" class="prompt">
	<label for="text-input">{prompt.message}</label>
	<div id="prompt-text-input-row" class="input-row">
		<input
			id="text-input"
			type="text"
			bind:value
			bind:this={inputEl}
			onkeydown={onKeydown}
		/>
		<button class="btn-primary" onclick={submit}>Submit</button>
	</div>
</div>

<style>
	.prompt { display: flex; flex-direction: column; gap: 0.5rem; }
	label { font-weight: 500; }
	.input-row { display: flex; gap: 0.5rem; }
	.input-row input { flex: 1; }
</style>
