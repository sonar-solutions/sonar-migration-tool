<script lang="ts">
	import '../app.css';
	import { onMount, onDestroy } from 'svelte';
	import { connect, disconnect, connectionState } from '$lib/stores/websocket';

	let { children } = $props();
	let connState = $derived($connectionState);

	onMount(() => connect());
	onDestroy(() => disconnect());
</script>

<header>
	<div class="container header-inner">
		<a href="/" class="logo">SonarQube Migration Tool</a>
		<nav>
			<a href="/">Wizard</a>
			<a href="/history">History</a>
		</nav>
		<div class="conn-status" class:open={connState === 'open'} class:closed={connState === 'closed'}>
			{connState === 'open' ? 'Connected' : connState === 'connecting' ? 'Connecting...' : 'Disconnected'}
		</div>
	</div>
</header>

<main class="container">
	{@render children()}
</main>

<style>
	header {
		background: white;
		border-bottom: 1px solid var(--color-border);
		padding: 0.75rem 0;
		margin-bottom: 1.5rem;
	}
	.header-inner {
		display: flex;
		align-items: center;
		gap: 1.5rem;
	}
	.logo {
		font-weight: 700;
		font-size: 1rem;
		color: var(--color-text);
		text-decoration: none;
	}
	nav {
		display: flex;
		gap: 1rem;
		flex: 1;
	}
	nav a {
		color: var(--color-text-muted);
		text-decoration: none;
		font-size: 0.9rem;
		padding: 0.25rem 0;
	}
	nav a:hover { color: var(--color-primary); }
	.conn-status {
		font-size: 0.75rem;
		padding: 0.25rem 0.625rem;
		border-radius: 999px;
		background: #fef2f2;
		color: var(--color-error);
	}
	.conn-status.open {
		background: #f0fdf4;
		color: var(--color-success);
	}
	main { padding-bottom: 2rem; }
</style>
