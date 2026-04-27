// Message types matching go/internal/gui/protocol.go

export interface KVPair {
	key: string;
	value: string;
}

export interface ServerMessage {
	type: string;
	id?: string;
	message?: string;
	validate?: boolean;
	default?: string | boolean;
	title?: string;
	details?: KVPair[];
	phase?: string;
	index?: number;
	total?: number;
	name?: string;
	stats?: KVPair[];
	source_url?: string;
	target_url?: string;
	extract_id?: string;
	error?: string | null;
}

export interface ClientMessage {
	type: string;
	id?: string;
	value?: string | boolean;
}

// Prompt types from server.
export const PROMPT_TYPES = [
	'prompt_url',
	'prompt_text',
	'prompt_password',
	'prompt_confirm',
	'prompt_confirm_review'
] as const;

export type PromptType = (typeof PROMPT_TYPES)[number];

export function isPromptMessage(msg: ServerMessage): boolean {
	return PROMPT_TYPES.includes(msg.type as PromptType);
}

// Run history types matching go/internal/gui/history.go
export interface RunInfo {
	run_id: string;
	source_url?: string;
	has_report: boolean;
	has_analysis: boolean;
}

export interface ReportRow {
	entity_type: string;
	entity_name: string;
	organization: string;
	url: string;
	http_status: string;
	outcome: string;
	error_message: string;
}

export interface WizardState {
	phase: string;
	extract_id?: string;
	source_url?: string;
	target_url?: string;
	enterprise_key?: string;
	organizations_mapped: boolean;
	validation_passed: boolean;
	migration_run_id?: string;
	skipped_projects?: string[];
}

// Phase display info.
export const PHASES = [
	{ key: 'extract', name: 'Extract', index: 1 },
	{ key: 'structure', name: 'Structure', index: 2 },
	{ key: 'org_mapping', name: 'Org Map', index: 3 },
	{ key: 'mappings', name: 'Mappings', index: 4 },
	{ key: 'validate', name: 'Validate', index: 5 },
	{ key: 'migrate', name: 'Migrate', index: 6 }
] as const;
