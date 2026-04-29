<script lang="ts">
	let dark = $state(true);

	$effect(() => {
		if (typeof window === 'undefined') {
			return;
		}
		const saved = localStorage.getItem('theme');
		dark = saved ? saved === 'dark' : true;
		document.documentElement.setAttribute('data-theme', dark ? 'dark' : 'light');
	});

	function toggle() {
		dark = !dark;
		document.documentElement.setAttribute('data-theme', dark ? 'dark' : 'light');
		localStorage.setItem('theme', dark ? 'dark' : 'light');
	}
</script>

<button class="theme-toggle" onclick={toggle} aria-label={dark ? 'Switch to light mode' : 'Switch to dark mode'}>
	{dark ? '\u2600' : '\u263D'}
</button>

<style>
	.theme-toggle {
		width: 2rem;
		height: 2rem;
		padding: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 1.1rem;
		background: var(--color-bg);
		border: 1px solid var(--color-border);
		border-radius: 50%;
		color: var(--color-text);
		cursor: pointer;
		transition: background-color 0.15s, border-color 0.15s;
	}
	.theme-toggle:hover {
		border-color: var(--color-primary);
	}
</style>
