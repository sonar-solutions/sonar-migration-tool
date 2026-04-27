<script lang="ts">
	import { respondToPrompt } from '$lib/stores/wizard';
	import type { ServerMessage } from '$lib/types';

	let { prompt }: { prompt: ServerMessage } = $props();
</script>

<div class="prompt">
	<p class="message">{prompt.message}</p>
	<div class="button-row">
		<button class="btn-primary" onclick={() => respondToPrompt(true)}>Yes</button>
		<button class="btn-secondary" onclick={() => respondToPrompt(false)}>No</button>
	</div>
</div>

<style>
	.prompt { display: flex; flex-direction: column; gap: 0.75rem; }
	.message { font-weight: 500; }
	.button-row { display: flex; gap: 0.5rem; }
</style>
