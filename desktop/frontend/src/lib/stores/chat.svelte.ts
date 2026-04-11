/**
 * Chat store — Svelte 5 runes-based.
 *
 * rAF batching: incoming "chat:chunk" events accumulate in a pending buffer.
 * A single requestAnimationFrame loop drains the buffer at frame rate.
 * This prevents reactive hangs when the gateway streams many chunks per second.
 */

// Wails runtime — auto-generated, won't exist until `wails generate module`
// Import path is relative to the wailsjs root at frontend/wailsjs
import { EventsOn } from '../../../wailsjs/runtime/runtime';

export interface Message {
	id: string;
	role: 'user' | 'assistant';
	content: string;
}

// ---------------------------------------------------------------------------
// Internal state (module-level $state via class pattern)
// ---------------------------------------------------------------------------

class ChatStore {
	messages = $state<Message[]>([]);
	currentStream = $state('');
	isStreaming = $state(false);
	sessionID = $state('dojo-desktop-' + Date.now());

	// rAF batching internals — NOT reactive, just plain fields
	private _pendingChunks: string[] = [];
	private _rafHandle: number | null = null;
	private _cleanup: (() => void) | null = null;

	constructor() {
		// Guard: EventsOn accesses `window` which doesn't exist during SSR.
		// +layout.ts sets ssr=false, but this guard is defense-in-depth.
		if (typeof window !== 'undefined') {
			this._startListening();
		}
	}

	private _startListening() {
		const off = EventsOn('chat:chunk', (payload: { content: string; done: boolean; error: string }) => {
			if (payload.error) {
				this._flushNow();
				this.isStreaming = false;
				return;
			}

			if (payload.content) {
				this._pendingChunks.push(payload.content);
				this._scheduleFlush();
			}

			if (payload.done) {
				// Flush remaining chunks, then finalize
				this._flushNow();
				this._finalizeStream();
			}
		});
		this._cleanup = off;
	}

	private _scheduleFlush() {
		if (this._rafHandle !== null) return; // already scheduled
		this._rafHandle = requestAnimationFrame(() => {
			this._rafHandle = null;
			this._drainPending();
		});
	}

	private _drainPending() {
		if (this._pendingChunks.length === 0) return;
		const batch = this._pendingChunks.splice(0);
		this.currentStream += batch.join('');
		// Schedule next drain if more chunks arrived while we were processing
		if (this._pendingChunks.length > 0) {
			this._scheduleFlush();
		}
	}

	private _flushNow() {
		if (this._rafHandle !== null) {
			cancelAnimationFrame(this._rafHandle);
			this._rafHandle = null;
		}
		this._drainPending();
	}

	private _finalizeStream() {
		if (this.currentStream.trim()) {
			this.messages = [
				...this.messages,
				{
					id: 'msg-' + Date.now(),
					role: 'assistant',
					content: this.currentStream
				}
			];
		}
		this.currentStream = '';
		this.isStreaming = false;
	}

	addUserMessage(content: string) {
		this.messages = [
			...this.messages,
			{
				id: 'msg-' + Date.now(),
				role: 'user',
				content
			}
		];
	}

	appendChunk(content: string, done: boolean) {
		// External callers can push chunks too (e.g., tests or direct invocations).
		// Uses the same rAF pipeline.
		if (content) {
			this._pendingChunks.push(content);
			this._scheduleFlush();
		}
		if (done) {
			this._flushNow();
			this._finalizeStream();
		}
	}

	clearStream() {
		this._flushNow();
		this.currentStream = '';
		this.isStreaming = false;
		this._pendingChunks = [];
	}

	destroy() {
		this._flushNow();
		if (this._cleanup) {
			this._cleanup();
			this._cleanup = null;
		}
	}
}

export const chatStore = new ChatStore();
