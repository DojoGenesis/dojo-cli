<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { chatStore } from '$lib/stores/chat.svelte';
	import { SendMessage, GetProviders, GetModels } from '../../../wailsjs/go/main/App';

	// Provider/model selectors
	// client.Provider: { name, status, info?, error? }  (status "active" = healthy)
	// client.Model:    { id, provider, name }
	let providers = $state<Array<{ name: string; status: string }>>([]);
	let models = $state<Array<{ id: string; provider: string; name: string }>>([]);
	let selectedProvider = $state('');
	let selectedModel = $state('');

	// Input state
	let inputText = $state('');
	let inputEl = $state<HTMLTextAreaElement | null>(null);
	let messagesEl = $state<HTMLDivElement | null>(null);

	// Derived — provider/model optional: gateway will use defaults if empty
	let canSend = $derived(
		inputText.trim().length > 0 &&
		!chatStore.isStreaming
	);

	onMount(async () => {
		try {
			const [p, m] = await Promise.all([GetProviders(), GetModels()]);
			providers = p.filter((x) => x.status === 'active');
			models = m;
			if (providers.length > 0) selectedProvider = providers[0].name;
			if (models.length > 0) selectedModel = models[0].id;
		} catch (e) {
			console.error('Failed to load providers/models:', e);
		}
	});

	// Auto-scroll to bottom when messages or stream changes
	$effect(() => {
		// Depend on both messages array length and currentStream
		const _ = chatStore.messages.length + chatStore.currentStream.length;
		tick().then(() => {
			if (messagesEl) {
				messagesEl.scrollTop = messagesEl.scrollHeight;
			}
		});
	});

	async function sendMessage() {
		const text = inputText.trim();
		if (!text || chatStore.isStreaming) return;

		inputText = '';
		chatStore.addUserMessage(text);
		chatStore.isStreaming = true;
		chatStore.currentStream = '';

		try {
			await SendMessage(chatStore.sessionID, text, selectedProvider, selectedModel);
		} catch (e) {
			chatStore.isStreaming = false;
			console.error('SendMessage failed:', e);
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			if (canSend) sendMessage();
		}
	}

	// Filter models by selected provider
	let filteredModels = $derived(
		selectedProvider ? models.filter((m) => m.provider === selectedProvider) : models
	);

	// Keep selectedModel valid when provider changes
	$effect(() => {
		if (filteredModels.length > 0) {
			const stillValid = filteredModels.some((m) => m.id === selectedModel);
			if (!stillValid) selectedModel = filteredModels[0].id;
		}
	});
</script>

<div class="chat-view">
	<!-- Toolbar -->
	<div class="toolbar">
		<div class="selectors">
			<select bind:value={selectedProvider} disabled={chatStore.isStreaming}>
				{#if providers.length === 0}
					<option value="">No providers</option>
				{:else}
					{#each providers as p}
						<option value={p.name}>{p.name}</option>
					{/each}
				{/if}
			</select>
			<select bind:value={selectedModel} disabled={chatStore.isStreaming}>
				{#if filteredModels.length === 0}
					<option value="">No models</option>
				{:else}
					{#each filteredModels as m}
						<option value={m.id}>{m.id}</option>
					{/each}
				{/if}
			</select>
		</div>
		{#if chatStore.isStreaming}
			<div class="streaming-indicator" aria-label="Streaming">
				<span class="dot"></span>
				<span class="dot"></span>
				<span class="dot"></span>
			</div>
		{/if}
	</div>

	<!-- Messages -->
	<div class="messages" bind:this={messagesEl}>
		{#if chatStore.messages.length === 0 && !chatStore.isStreaming}
			<div class="empty-state">
				<span class="empty-icon">◈</span>
				<p>Start a conversation</p>
			</div>
		{/if}

		{#each chatStore.messages as msg (msg.id)}
			<div class="message {msg.role}">
				<div class="message-role">{msg.role === 'user' ? 'You' : 'Dojo'}</div>
				<div class="message-body">
					<pre class="message-text">{msg.content}</pre>
				</div>
			</div>
		{/each}

		{#if chatStore.isStreaming || chatStore.currentStream}
			<div class="message assistant streaming">
				<div class="message-role">Dojo</div>
				<div class="message-body">
					{#if chatStore.currentStream}
						<pre class="message-text">{chatStore.currentStream}</pre>
					{:else}
						<div class="typing-dots">
							<span class="dot"></span>
							<span class="dot"></span>
							<span class="dot"></span>
						</div>
					{/if}
				</div>
			</div>
		{/if}
	</div>

	<!-- Input area -->
	<div class="input-area">
		<textarea
			bind:this={inputEl}
			bind:value={inputText}
			onkeydown={handleKeydown}
			placeholder="Message Dojo… (Enter to send, Shift+Enter for newline)"
			rows={3}
			disabled={chatStore.isStreaming}
		></textarea>
		<button
			class="send-btn"
			onclick={sendMessage}
			disabled={!canSend}
			aria-label="Send message"
		>
			Send
		</button>
	</div>
</div>

<style>
	.chat-view {
		display: flex;
		flex-direction: column;
		height: 100%;
		overflow: hidden;
	}

	/* Toolbar */
	.toolbar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 8px 12px;
		border-bottom: 1px solid var(--border);
		background: var(--surface);
		flex-shrink: 0;
		gap: 8px;
	}

	.selectors {
		display: flex;
		gap: 6px;
	}

	select {
		padding: 4px 8px;
		font-size: 12px;
		border-radius: var(--radius);
		color: var(--text);
		background: var(--bg);
		border: 1px solid var(--border);
		cursor: pointer;
	}

	select:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	/* Streaming indicator in toolbar */
	.streaming-indicator {
		display: flex;
		gap: 3px;
		align-items: center;
	}

	/* Messages */
	.messages {
		flex: 1;
		overflow-y: auto;
		padding: 16px 12px;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.empty-state {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 8px;
		color: var(--muted);
		margin: auto;
		padding: 40px;
	}

	.empty-icon {
		font-size: 32px;
		color: var(--accent);
	}

	.message {
		display: flex;
		flex-direction: column;
		gap: 4px;
		max-width: 820px;
	}

	.message.user {
		align-self: flex-end;
	}

	.message.assistant {
		align-self: flex-start;
	}

	.message-role {
		font-size: 11px;
		font-weight: 600;
		color: var(--muted);
		text-transform: uppercase;
		letter-spacing: 0.05em;
		padding: 0 4px;
	}

	.message.user .message-role {
		text-align: right;
		color: color-mix(in srgb, var(--accent) 80%, var(--muted));
	}

	.message-body {
		padding: 10px 14px;
		border-radius: var(--radius);
		background: var(--surface);
		border: 1px solid var(--border);
	}

	.message.user .message-body {
		background: color-mix(in srgb, var(--accent) 12%, var(--surface));
		border-color: color-mix(in srgb, var(--accent) 30%, var(--border));
	}

	.message-text {
		white-space: pre-wrap;
		word-break: break-word;
		font-family: inherit;
		font-size: 14px;
		line-height: 1.6;
		color: var(--text);
		margin: 0;
	}

	.message.streaming .message-body {
		border-color: color-mix(in srgb, var(--accent) 40%, var(--border));
	}

	/* Animated dots */
	.typing-dots,
	.streaming-indicator {
		display: flex;
		gap: 4px;
		align-items: center;
		padding: 2px 0;
	}

	.dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: var(--accent);
		animation: bounce 1.2s ease-in-out infinite;
	}

	.dot:nth-child(2) {
		animation-delay: 0.2s;
	}

	.dot:nth-child(3) {
		animation-delay: 0.4s;
	}

	@keyframes bounce {
		0%, 60%, 100% {
			transform: translateY(0);
			opacity: 0.4;
		}
		30% {
			transform: translateY(-5px);
			opacity: 1;
		}
	}

	/* Input area */
	.input-area {
		display: flex;
		gap: 8px;
		padding: 10px 12px;
		border-top: 1px solid var(--border);
		background: var(--surface);
		flex-shrink: 0;
		align-items: flex-end;
	}

	textarea {
		flex: 1;
		resize: none;
		padding: 8px 12px;
		font-size: 14px;
		line-height: 1.5;
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--bg);
		color: var(--text);
		transition: border-color 0.15s;
	}

	textarea:focus {
		border-color: var(--accent);
	}

	textarea:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	textarea::placeholder {
		color: var(--muted);
	}

	.send-btn {
		padding: 8px 18px;
		border-radius: var(--radius);
		background: var(--accent);
		color: #fff;
		font-size: 13px;
		font-weight: 600;
		transition: background 0.15s, opacity 0.15s;
		white-space: nowrap;
		height: fit-content;
	}

	.send-btn:hover:not(:disabled) {
		background: var(--accent-hover);
	}

	.send-btn:disabled {
		opacity: 0.4;
		cursor: not-allowed;
	}
</style>
