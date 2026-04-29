import { writable, readonly } from 'svelte/store';
import type { ServerMessage, ClientMessage } from '$lib/types';

export type ConnectionState = 'connecting' | 'open' | 'closed';

const _connectionState = writable<ConnectionState>('closed');
export const connectionState = readonly(_connectionState);

const _lastMessage = writable<ServerMessage | null>(null);
export const lastMessage = readonly(_lastMessage);

let ws: WebSocket | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let reconnectDelay = 1000;

export function connect() {
	if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
		return;
	}

	const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
	const url = `${protocol}//${location.host}/ws`;

	_connectionState.set('connecting');
	ws = new WebSocket(url);

	ws.onopen = () => {
		_connectionState.set('open');
		reconnectDelay = 1000; // Reset backoff on successful connect.
	};

	ws.onclose = () => {
		_connectionState.set('closed');
		ws = null;
		scheduleReconnect();
	};

	ws.onerror = () => {
		// onclose will fire after onerror.
	};

	ws.onmessage = (event) => {
		try {
			const msg: ServerMessage = JSON.parse(event.data);
			_lastMessage.set(msg);
		} catch {
			console.error('Failed to parse WebSocket message:', event.data);
		}
	};
}

export function disconnect() {
	if (reconnectTimer) {
		clearTimeout(reconnectTimer);
		reconnectTimer = null;
	}
	if (ws) {
		ws.close();
		ws = null;
	}
	_connectionState.set('closed');
}

export function send(msg: ClientMessage) {
	if (ws && ws.readyState === WebSocket.OPEN) {
		ws.send(JSON.stringify(msg));
	}
}

function scheduleReconnect() {
	if (reconnectTimer) return;
	reconnectTimer = setTimeout(() => {
		reconnectTimer = null;
		reconnectDelay = Math.min(reconnectDelay * 2, 10000);
		connect();
	}, reconnectDelay);
}
