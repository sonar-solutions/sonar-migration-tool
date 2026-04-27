import { writable, readonly, derived } from 'svelte/store';
import { lastMessage, send } from './websocket';
import type { ServerMessage } from '$lib/types';
import { isPromptMessage } from '$lib/types';

export type WizardStatus = 'idle' | 'running' | 'finished' | 'error';

// Current wizard status.
const _status = writable<WizardStatus>('idle');
export const wizardStatus = readonly(_status);

// Active phase info.
const _currentPhase = writable<{ phase: string; index: number; total: number; name: string } | null>(null);
export const currentPhase = readonly(_currentPhase);

// Pending prompt (only one at a time since wizard is synchronous).
const _pendingPrompt = writable<ServerMessage | null>(null);
export const pendingPrompt = readonly(_pendingPrompt);

// Event log — all display messages.
export interface LogEntry {
	type: string;
	message: string;
	timestamp: Date;
}
const _eventLog = writable<LogEntry[]>([]);
export const eventLog = readonly(_eventLog);

// Subscribe to incoming WebSocket messages and dispatch.
lastMessage.subscribe((msg) => {
	if (!msg) return;
	handleMessage(msg);
});

function handleMessage(msg: ServerMessage) {
	// Prompts.
	if (isPromptMessage(msg)) {
		_pendingPrompt.set(msg);
		return;
	}

	switch (msg.type) {
		case 'wizard_started':
			_status.set('running');
			_eventLog.set([]);
			_currentPhase.set(null);
			break;

		case 'wizard_finished':
			_status.set(msg.error ? 'error' : 'finished');
			if (msg.error) {
				addLog('display_error', msg.error);
			}
			break;

		case 'display_phase_progress':
			_currentPhase.set({
				phase: msg.phase || '',
				index: msg.index || 0,
				total: msg.total || 6,
				name: msg.name || ''
			});
			addLog(msg.type, `Phase ${msg.index}/${msg.total}: ${msg.name}`);
			break;

		case 'display_welcome':
			addLog(msg.type, 'Welcome to the SonarQube Migration Wizard');
			break;

		case 'display_wizard_complete':
			addLog(msg.type, 'Wizard complete! Your migration is finished.');
			break;

		case 'display_message':
		case 'display_error':
		case 'display_warning':
		case 'display_success':
			addLog(msg.type, msg.message || '');
			break;

		case 'display_summary':
			if (msg.stats) {
				const lines = msg.stats.map((s) => `${s.key}: ${s.value}`).join(', ');
				addLog(msg.type, `${msg.title} — ${lines}`);
			}
			break;

		case 'display_resume_info':
			addLog(msg.type, `Previous session: phase=${msg.phase}, source=${msg.source_url || 'N/A'}`);
			break;
	}
}

function addLog(type: string, message: string) {
	_eventLog.update((log) => [...log, { type, message, timestamp: new Date() }]);
}

// Actions.
export function startWizard() {
	_status.set('idle');
	_eventLog.set([]);
	_currentPhase.set(null);
	_pendingPrompt.set(null);
	send({ type: 'start_wizard' });
}

export function cancelWizard() {
	send({ type: 'cancel_wizard' });
}

export function respondToPrompt(value: string | boolean) {
	let currentPrompt: ServerMessage | null = null;
	_pendingPrompt.subscribe((p) => (currentPrompt = p))();

	if (currentPrompt && currentPrompt.id) {
		send({ type: 'prompt_response', id: currentPrompt.id, value });
		_pendingPrompt.set(null);
	}
}
