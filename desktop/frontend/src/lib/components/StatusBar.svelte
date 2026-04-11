<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { chatStore } from '$lib/stores/chat.svelte';
	import { CheckHealth } from '../../../wailsjs/go/main/App';

	let connected = $state(false);
	let checking = $state(true);
	let pollInterval: ReturnType<typeof setInterval> | null = null;

	async function checkHealth() {
		try {
			connected = await CheckHealth();
		} catch {
			connected = false;
		} finally {
			checking = false;
		}
	}

	onMount(() => {
		checkHealth();
		pollInterval = setInterval(checkHealth, 30_000);
	});

	onDestroy(() => {
		if (pollInterval !== null) {
			clearInterval(pollInterval);
		}
	});

	let statusLabel = $derived(
		checking ? 'Checking…' : connected ? 'Gateway connected' : 'Gateway offline'
	);
</script>

<div class="status-bar">
	<div class="status-left">
		<span
			class="indicator"
			class:green={connected && !checking}
			class:red={!connected && !checking}
			class:dim={checking}
			aria-hidden="true"
		></span>
		<span class="status-text">{statusLabel}</span>
	</div>

	<div class="status-right">
		<span class="session-label">Session:</span>
		<span class="session-id">{chatStore.sessionID}</span>
	</div>
</div>

<style>
	.status-bar {
		height: 28px;
		background: color-mix(in srgb, var(--surface) 90%, var(--bg));
		border-top: 1px solid var(--border);
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0 12px;
		flex-shrink: 0;
	}

	.status-left,
	.status-right {
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.indicator {
		width: 7px;
		height: 7px;
		border-radius: 50%;
		background: var(--muted);
		flex-shrink: 0;
		transition: background 0.3s;
	}

	.indicator.green {
		background: #4ade80;
		box-shadow: 0 0 4px #4ade8066;
	}

	.indicator.red {
		background: #f87171;
	}

	.indicator.dim {
		background: var(--muted);
		opacity: 0.5;
	}

	.status-text {
		font-size: 11px;
		color: var(--muted);
	}

	.session-label {
		font-size: 11px;
		color: var(--muted);
		opacity: 0.6;
	}

	.session-id {
		font-size: 11px;
		color: var(--muted);
		font-variant-numeric: tabular-nums;
	}
</style>
