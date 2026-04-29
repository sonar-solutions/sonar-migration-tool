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

// Latest display message (shown alongside spinner when processing).
const _latestDisplayMessage = writable<string>('');
export const latestDisplayMessage = readonly(_latestDisplayMessage);

// True when wizard is running but no prompt is pending (backend is working).
export const isProcessing = derived(
	[_status, _pendingPrompt],
	([$s, $p]) => $s === 'running' && $p === null
);

// Latest summary data (shown alongside confirm prompts).
export interface SummaryData {
	title: string;
	stats: Array<{ key: string; value: string }>;
}
const _latestSummary = writable<SummaryData | null>(null);
export const latestSummary = readonly(_latestSummary);

// Collected user inputs per phase (displayed in the phases panel).
export interface PhaseInput {
	label: string;
	value: string;
}
const _phaseInputs = writable<Record<string, PhaseInput[]>>({});
export const phaseInputs = readonly(_phaseInputs);

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
	if (!msg) {
		return;
	}
	handleMessage(msg);
});

function handleMessage(msg: ServerMessage) {
	// Prompts.
	if (isPromptMessage(msg)) {
		_pendingPrompt.set(msg);
		_latestDisplayMessage.set('');
		return;
	}

	switch (msg.type) {
		case 'wizard_started':
			_status.set('running');
			_eventLog.set([]);
			_currentPhase.set(null);
			_latestDisplayMessage.set('');
			_phaseInputs.set({});
			_latestSummary.set(null);
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
			_latestDisplayMessage.set(`Phase ${msg.index}/${msg.total}: ${msg.name}`);
			addLog(msg.type, `Phase ${msg.index}/${msg.total}: ${msg.name}`);
			break;

		case 'display_welcome':
			addLog(msg.type, 'Welcome to the SonarQube Migration Wizard');
			break;

		case 'display_wizard_complete':
			addLog(msg.type, 'Wizard complete! Your migration is finished.');
			break;

		case 'display_message':
			_latestDisplayMessage.set(msg.message || '');
			addLog(msg.type, msg.message || '');
			break;

		case 'display_error':
		case 'display_warning':
		case 'display_success':
			addLog(msg.type, msg.message || '');
			break;

		case 'display_summary':
			if (msg.stats) {
				_latestSummary.set({
					title: msg.title || '',
					stats: msg.stats.map((s) => ({ key: s.key, value: s.value }))
				});
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
		capturePhaseInput(currentPrompt, value);
		send({ type: 'prompt_response', id: currentPrompt.id, value });
		_pendingPrompt.set(null);
		_latestSummary.set(null);
	}
}

function capturePhaseInput(prompt: ServerMessage, value: string | boolean) {
	let phase: { phase: string } | null = null;
	_currentPhase.subscribe((p) => (phase = p))();
	if (!phase) {
		return;
	}

	const label = prompt.message || prompt.title || '';
	if (!label) {
		return;
	}

	let displayValue: string;
	if (prompt.type === 'prompt_password') {
		displayValue = '********';
	} else if (typeof value === 'boolean') {
		displayValue = value ? 'Yes' : 'No';
	} else {
		displayValue = value;
	}

	const key = phase.phase;
	_phaseInputs.update((inputs) => {
		const existing = inputs[key] || [];
		return { ...inputs, [key]: [...existing, { label, value: displayValue }] };
	});
}
