<script lang="ts">
	import { marked } from 'marked';

	let { markdown }: { markdown: string } = $props();
	let html = $derived(marked(markdown));
</script>

<div class="report-content">
	{@html html}
</div>

<style>
	.report-content {
		line-height: 1.7;
		font-size: 0.9rem;
	}
	.report-content :global(h1) { font-size: 1.5rem; margin: 1.5rem 0 0.75rem; }
	.report-content :global(h2) { font-size: 1.25rem; margin: 1.25rem 0 0.5rem; }
	.report-content :global(h3) { font-size: 1.1rem; margin: 1rem 0 0.5rem; }
	.report-content :global(p) { margin: 0.5rem 0; }
	.report-content :global(table) {
		width: 100%;
		border-collapse: collapse;
		margin: 0.75rem 0;
		font-size: 0.85rem;
	}
	.report-content :global(th),
	.report-content :global(td) {
		padding: 0.375rem 0.625rem;
		border: 1px solid var(--color-border);
		text-align: left;
	}
	.report-content :global(th) {
		background: var(--color-bg);
		font-weight: 600;
	}
	.report-content :global(code) {
		background: var(--color-bg);
		padding: 0.125rem 0.375rem;
		border-radius: 3px;
		font-size: 0.85em;
	}
	.report-content :global(pre) {
		background: #f1f3f5;
		padding: 0.75rem 1rem;
		border-radius: var(--radius);
		overflow-x: auto;
	}
	.report-content :global(ul),
	.report-content :global(ol) {
		padding-left: 1.5rem;
		margin: 0.5rem 0;
	}
</style>
