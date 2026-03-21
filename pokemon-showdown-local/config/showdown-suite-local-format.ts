import * as fs from 'fs';
import * as path from 'path';
import {Pokedex} from '../data/pokedex';

const LOCAL_FORMAT_NAME = '[Gen 9] Showdown Suite Studio';
const LOCAL_FORMAT_ID = 'gen9showdownsuitestudio';
function getConfigPath() {
	const candidates = [
		path.resolve(__dirname, 'showdown-suite-local-format.json'),
		path.resolve(__dirname, '..', '..', 'config', 'showdown-suite-local-format.json'),
	];
	for (const candidate of candidates) {
		if (fs.existsSync(candidate)) return candidate;
	}
	return candidates[0];
}

type StudioPresetID = 'standard-singles' | 'standard-doubles' | 'custom-singles' | 'custom-doubles';

interface StudioPreset {
	id: StudioPresetID;
	label: string;
	gameType: 'singles' | 'doubles';
	ruleset: string[];
	summary: string;
	defaultLevel: number;
	defaultMaxTeamSize: number;
	defaultPickedTeamSize: number;
}

interface StudioFormatConfig {
	preset: StudioPresetID;
	level: number;
	maxTeamSize: number;
	pickedTeamSize: number;
	openTeamSheets: boolean;
	allowTerastal: boolean;
	customRules: string[];
	targetPokemon: string[];
	bannedPokemon: string[];
}

const PRESETS: Record<StudioPresetID, StudioPreset> = {
	'standard-singles': {
		id: 'standard-singles',
		label: 'Standard Singles',
		gameType: 'singles',
		ruleset: ['Standard'],
		summary: 'Smogon-style singles baseline with local target-pool enforcement.',
		defaultLevel: 50,
		defaultMaxTeamSize: 3,
		defaultPickedTeamSize: 3,
	},
	'standard-doubles': {
		id: 'standard-doubles',
		label: 'Standard Doubles',
		gameType: 'doubles',
		ruleset: ['Standard Doubles'],
		summary: 'Smogon-style doubles baseline with local target-pool enforcement.',
		defaultLevel: 50,
		defaultMaxTeamSize: 4,
		defaultPickedTeamSize: 4,
	},
	'custom-singles': {
		id: 'custom-singles',
		label: 'Custom Singles',
		gameType: 'singles',
		ruleset: ['Standard AG'],
		summary: 'Lightweight custom singles baseline with team preview and local validation.',
		defaultLevel: 50,
		defaultMaxTeamSize: 3,
		defaultPickedTeamSize: 3,
	},
	'custom-doubles': {
		id: 'custom-doubles',
		label: 'Custom Doubles',
		gameType: 'doubles',
		ruleset: ['Standard AG'],
		summary: 'Lightweight custom doubles baseline with team preview and local validation.',
		defaultLevel: 50,
		defaultMaxTeamSize: 4,
		defaultPickedTeamSize: 4,
	},
};

const DEFAULT_CONFIG: StudioFormatConfig = {
	preset: 'standard-singles',
	level: 50,
	maxTeamSize: 3,
	pickedTeamSize: 3,
	openTeamSheets: false,
	allowTerastal: true,
	customRules: [],
	targetPokemon: ['Pikachu', 'Charizard', 'Meowscarada'],
	bannedPokemon: [],
};

function toID(text: string) {
	return ('' + text).toLowerCase().replace(/[^a-z0-9]+/g, '');
}

function clampInt(value: unknown, fallback: number, min: number, max: number) {
	const numeric = Number(value);
	if (!Number.isFinite(numeric)) return fallback;
	return Math.max(min, Math.min(max, Math.trunc(numeric)));
}

function sanitizeSpeciesList(value: unknown, fieldName: string) {
	if (!Array.isArray(value)) return [];
	const seen = new Set<string>();
	const out = [];
	for (const raw of value) {
		const name = `${raw ?? ''}`.trim();
		if (!name) continue;
		const species = Pokedex[toID(name)];
		if (!species?.name) {
			throw new Error(`Unknown species in ${fieldName}: ${name}`);
		}
		const id = toID(species.name);
		if (!id || seen.has(id)) continue;
		seen.add(id);
		out.push(species.name);
	}
	return out;
}

function sanitizeRuleList(value: unknown) {
	if (!Array.isArray(value)) return [];
	const seen = new Set<string>();
	const out = [];
	for (const raw of value) {
		const rule = `${raw ?? ''}`.trim();
		if (!rule) continue;
		const dedupeKey = normalizeRuleKey(rule);
		if (seen.has(dedupeKey)) continue;
		seen.add(dedupeKey);
		out.push(rule);
	}
	return out;
}

function normalizeRuleKey(value: string) {
	return value.toLowerCase().replace(/\s+/g, '');
}

function filterCustomRules(config: StudioFormatConfig) {
	const baseRules = new Set<string>([
		...PRESETS[config.preset].ruleset.map(normalizeRuleKey),
		normalizeRuleKey(`Adjust Level = ${config.level}`),
		normalizeRuleKey(`Max Team Size = ${config.maxTeamSize}`),
		normalizeRuleKey(`Picked Team Size = ${config.pickedTeamSize}`),
	]);
	if (config.openTeamSheets) baseRules.add(normalizeRuleKey('Open Team Sheets'));
	if (!config.allowTerastal) baseRules.add(normalizeRuleKey('Terastal Clause'));
	return {
		...config,
		customRules: config.customRules.filter(rule => !baseRules.has(normalizeRuleKey(rule))),
	};
}

function readConfigFile() {
	try {
		return JSON.parse(fs.readFileSync(getConfigPath(), 'utf8'));
	} catch {
		return {};
	}
}

export function getStudioPresets() {
	return Object.values(PRESETS).map(preset => ({
		id: preset.id,
		label: preset.label,
		mode: preset.gameType,
		summary: preset.summary,
		defaultLevel: preset.defaultLevel,
		defaultMaxTeamSize: preset.defaultMaxTeamSize,
		defaultPickedTeamSize: preset.defaultPickedTeamSize,
	}));
}

export function readStudioFormatConfig(): StudioFormatConfig {
	const raw = { ...DEFAULT_CONFIG, ...readConfigFile() };
	const preset = PRESETS[(raw.preset as StudioPresetID)] ? raw.preset as StudioPresetID : DEFAULT_CONFIG.preset;
	const presetMeta = PRESETS[preset];
	const minTeamSize = presetMeta.gameType === 'doubles' ? 2 : 1;
	const maxTeamSize = clampInt(
		raw.maxTeamSize,
		presetMeta.defaultMaxTeamSize,
		minTeamSize,
		6
	);
	return filterCustomRules({
		preset,
		level: clampInt(raw.level, presetMeta.defaultLevel, 1, 100),
		maxTeamSize,
		pickedTeamSize: clampInt(raw.pickedTeamSize, presetMeta.defaultPickedTeamSize, minTeamSize, maxTeamSize),
		openTeamSheets: !!raw.openTeamSheets,
		allowTerastal: raw.allowTerastal !== false,
		customRules: sanitizeRuleList(raw.customRules),
		targetPokemon: sanitizeSpeciesList(raw.targetPokemon, 'targetPokemon'),
		bannedPokemon: sanitizeSpeciesList(raw.bannedPokemon, 'bannedPokemon'),
	});
}

function buildRuleset(config: StudioFormatConfig) {
	const preset = PRESETS[config.preset];
	const ruleset = [...preset.ruleset];
	ruleset.push(`Adjust Level = ${config.level}`);
	ruleset.push(`Max Team Size = ${config.maxTeamSize}`);
	ruleset.push(`Picked Team Size = ${config.pickedTeamSize}`);
	if (config.openTeamSheets) ruleset.push('Open Team Sheets');
	if (!config.allowTerastal) ruleset.push('Terastal Clause');
	ruleset.push(...config.customRules);
	return ruleset;
}

function buildSummary(config: StudioFormatConfig) {
	const preset = PRESETS[config.preset];
	return [
		`${preset.label} preset`,
		`Level: ${config.level}`,
		`Bring ${config.maxTeamSize}, choose ${config.pickedTeamSize}`,
		config.openTeamSheets ? 'Open Team Sheets enabled' : 'Open Team Sheets disabled',
		config.allowTerastal ? 'Terastallization allowed' : 'Terastallization disabled',
		config.customRules.length ?
			`Custom rules: ${config.customRules.join(', ')}` :
			'Custom rules: none',
		config.targetPokemon.length ?
			`Target Pokemon: ${config.targetPokemon.join(', ')}` :
			'Target Pokemon: unrestricted',
		config.bannedPokemon.length ?
			`Additional banned Pokemon: ${config.bannedPokemon.join(', ')}` :
			'Additional banned Pokemon: none',
	];
}

export function getStudioFormatQueryData() {
	const config = readStudioFormatConfig();
	const preset = PRESETS[config.preset];
	return {
		formatId: LOCAL_FORMAT_ID,
		name: LOCAL_FORMAT_NAME,
		config,
		preset: {
			id: preset.id,
			label: preset.label,
			mode: preset.gameType,
			summary: preset.summary,
		},
		ruleset: buildRuleset(config),
		summary: buildSummary(config),
		targetPokemon: [...config.targetPokemon],
		bannedPokemon: [...config.bannedPokemon],
		presets: getStudioPresets(),
	};
}

export function buildLocalStudioFormat(): import('../sim/dex-formats').FormatData {
	const config = readStudioFormatConfig();
	const preset = PRESETS[config.preset];
	const targetIDs = new Map(config.targetPokemon.map(name => [toID(name), name]));
	const bannedIDs = new Map(config.bannedPokemon.map(name => [toID(name), name]));
	const desc = buildSummary(config).join('<br />');

	return {
		name: LOCAL_FORMAT_NAME,
		mod: 'gen9',
		gameType: preset.gameType === 'doubles' ? 'doubles' : undefined,
		searchShow: false,
		rated: false,
		threads: [
			`Use the local Showdown Suite GUI or edit <code>${getConfigPath()}</code> to change this format.`,
		],
		ruleset: buildRuleset(config),
		desc,
		onValidateSet(set) {
			const displayName = set.species || set.name;
			const speciesId = toID(set.species);
			if (bannedIDs.has(speciesId)) {
				return [
					`${displayName} is banned by ${LOCAL_FORMAT_NAME}.`,
					`Additional banned Pokemon: ${config.bannedPokemon.join(', ')}`,
				];
			}
			if (targetIDs.size && !targetIDs.has(speciesId)) {
				return [
					`${displayName} is outside the current target Pokemon pool for ${LOCAL_FORMAT_NAME}.`,
					`Allowed Pokemon: ${config.targetPokemon.join(', ')}`,
				];
			}
		},
	};
}
