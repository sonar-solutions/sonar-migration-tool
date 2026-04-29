<script lang="ts">
	import '../app.css';
	import { onMount, onDestroy } from 'svelte';
	import { connect, disconnect, connectionState } from '$lib/stores/websocket';
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';

	let { children } = $props();
	let connState = $derived($connectionState);

	onMount(() => connect());
	onDestroy(() => disconnect());
</script>

<header>
	<div id="header-inner" class="container header-inner">
		<a href="/" class="logo">SonarQube Migration Tool</a>
		<nav>
			<a href="/">Wizard</a>
			<a href="/history">History</a>
		</nav>
		<div id="connection-status" class="conn-status" class:open={connState === 'open'} class:closed={connState === 'closed'}>
			{connState === 'open' ? 'Connected' : connState === 'connecting' ? 'Connecting...' : 'Disconnected'}
		</div>
		<ThemeToggle />
	</div>
</header>

<main class="container">
	{@render children()}
</main>

<style>
	header {
		background: var(--color-surface);
		border-bottom: 1px solid var(--color-border);
		padding: 0.75rem 0;
		margin-bottom: 1.5rem;
		transition: background-color 0.2s;
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
		background: var(--color-badge-error-bg);
		color: var(--color-error);
	}
	.conn-status.open {
		background: var(--color-badge-success-bg);
		color: var(--color-success);
	}
	main { padding-bottom: 2rem; }
</style>
