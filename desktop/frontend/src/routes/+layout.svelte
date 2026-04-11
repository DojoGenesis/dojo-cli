<script lang="ts">
	import '../app.css';
	import StatusBar from '$lib/components/StatusBar.svelte';

	let { children } = $props();
</script>

<div class="shell">
	<aside class="sidebar">
		<div class="sidebar-header">
			<span class="logo-mark">◈</span>
			<span class="logo-name">Dojo</span>
		</div>
		<nav class="sidebar-nav">
			<a href="/" class="nav-item active">
				<span class="nav-icon">💬</span>
				<span>Chat</span>
			</a>
		</nav>
		<div class="sidebar-footer">
			<span class="version">v0.1</span>
		</div>
	</aside>

	<main class="main">
		{@render children()}
	</main>
</div>

<StatusBar />

<style>
	.shell {
		display: flex;
		height: calc(100vh - 28px); /* leave room for status bar */
		width: 100%;
		overflow: hidden;
	}

	.sidebar {
		width: var(--sidebar-width);
		flex-shrink: 0;
		background: var(--surface);
		border-right: 1px solid var(--border);
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.sidebar-header {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 16px 16px 12px;
		border-bottom: 1px solid var(--border);
	}

	.logo-mark {
		font-size: 18px;
		color: var(--accent);
		line-height: 1;
	}

	.logo-name {
		font-size: 16px;
		font-weight: 600;
		letter-spacing: 0.02em;
		color: var(--text);
	}

	.sidebar-nav {
		flex: 1;
		padding: 8px;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.nav-item {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 10px;
		border-radius: var(--radius);
		color: var(--muted);
		font-size: 13px;
		transition: background 0.1s, color 0.1s;
		text-decoration: none;
	}

	.nav-item:hover {
		background: var(--border);
		color: var(--text);
	}

	.nav-item.active {
		background: color-mix(in srgb, var(--accent) 15%, transparent);
		color: var(--accent-hover);
	}

	.nav-icon {
		font-size: 14px;
		line-height: 1;
	}

	.sidebar-footer {
		padding: 10px 16px;
		border-top: 1px solid var(--border);
	}

	.version {
		font-size: 11px;
		color: var(--muted);
	}

	.main {
		flex: 1;
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}
</style>
