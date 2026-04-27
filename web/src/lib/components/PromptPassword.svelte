<script lang="ts">
	import { respondToPrompt } from '$lib/stores/wizard';
	import type { ServerMessage } from '$lib/types';

	let { prompt }: { prompt: ServerMessage } = $props();
	let value = $state('');
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

<div class="prompt">
	<label for="password-input">{prompt.message}</label>
	<div class="input-row">
		<input
			id="password-input"
			type="password"
			bind:value
			bind:this={inputEl}
			onkeydown={onKeydown}
			autocomplete="off"
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
