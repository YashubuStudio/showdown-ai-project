#!/usr/bin/env node
'use strict';

const fs = require('fs');
const path = require('path');
const https = require('https');
const {resolveShowdownRoot, showdownDistPath} = require('./showdown-source');

const rootDir = path.resolve(__dirname, '..');
const dataDir = path.join(rootDir, 'play.pokemonshowdown.com/data');
const cacheDir = path.join(rootDir, 'caches');
const outputPath = path.join(dataDir, 'translations-ja.js');
const cachePath = path.join(cacheDir, 'pokeapi-ja-cache.json');
const showdownRoot = resolveShowdownRoot();

const csvBase = 'https://raw.githubusercontent.com/veekun/pokedex/master/pokedex/data/csv/';
const pokeApiBase = 'https://pokeapi.co/api/v2/';
const preferredLanguageIds = ['11', '1'];
const preferredLanguageNames = ['ja', 'ja-Hrkt'];
const apiHeaders = {
	'User-Agent': 'showdown-suite-local/1.0',
	'Accept': 'application/json,text/plain,*/*',
};

function canonicalId(text) {
	return `${text || ''}`.toLowerCase().replace(/[^a-z0-9]+/g, '');
}

function parseCSV(text) {
	if (text.charCodeAt(0) === 0xFEFF) text = text.slice(1);
	const rows = [];
	let row = [];
	let field = '';
	let inQuotes = false;

	for (let i = 0; i < text.length; i++) {
		const ch = text[i];
		if (inQuotes) {
			if (ch === '"') {
				if (text[i + 1] === '"') {
					field += '"';
					i++;
				} else {
					inQuotes = false;
				}
			} else {
				field += ch;
			}
			continue;
		}
		if (ch === '"') {
			inQuotes = true;
		} else if (ch === ',') {
			row.push(field);
			field = '';
		} else if (ch === '\n') {
			row.push(field);
			rows.push(row);
			row = [];
			field = '';
		} else if (ch !== '\r') {
			field += ch;
		}
	}
	if (field || row.length) {
		row.push(field);
		rows.push(row);
	}
	const [header, ...body] = rows;
	return body
		.filter(columns => columns.length && columns.some(value => value !== ''))
		.map(columns => Object.fromEntries(header.map((key, index) => [key, columns[index] || ''])));
}

function fetchText(url) {
	return new Promise((resolve, reject) => {
		https.get(url, {headers: apiHeaders}, response => {
			if (response.statusCode >= 300 && response.statusCode < 400 && response.headers.location) {
				response.resume();
				resolve(fetchText(response.headers.location));
				return;
			}
			if (response.statusCode !== 200) {
				reject(new Error(`Failed to fetch ${url}: ${response.statusCode}`));
				response.resume();
				return;
			}
			let data = '';
			response.setEncoding('utf8');
			response.on('data', chunk => {
				data += chunk;
			});
			response.on('end', () => resolve(data));
		}).on('error', reject);
	});
}

async function fetchCSV(name) {
	return parseCSV(await fetchText(`${csvBase}${name}`));
}

async function fetchJSON(url) {
	return JSON.parse(await fetchText(url));
}

function loadCache() {
	try {
		return JSON.parse(fs.readFileSync(cachePath, 'utf8'));
	} catch {
		return {};
	}
}

function saveCache(cache) {
	fs.mkdirSync(cacheDir, {recursive: true});
	fs.writeFileSync(cachePath, JSON.stringify(cache, null, 2));
}

function isEnglishish(name) {
	return !name || /[A-Za-z]/.test(name);
}

function mapNamesById(rows, idKey) {
	const byId = {};
	for (const row of rows) {
		const priority = preferredLanguageIds.indexOf(row.local_language_id);
		if (priority < 0 || !row.name) continue;
		const current = byId[row[idKey]];
		if (!current || priority < current.priority) {
			byId[row[idKey]] = {name: row.name, priority};
		}
	}
	return Object.fromEntries(Object.entries(byId).map(([id, value]) => [id, value.name]));
}

function extractLocalizedName(entries) {
	for (const language of preferredLanguageNames) {
		const match = entries?.find(entry => entry.language?.name === language && entry.name);
		if (match) return match.name;
	}
	return '';
}

async function fetchCachedPokeApiName(endpoint, extractor, cache) {
	if (cache[endpoint] !== undefined) return cache[endpoint];
	try {
		const data = await fetchJSON(`${pokeApiBase}${endpoint}/`);
		cache[endpoint] = extractor(data) || null;
	} catch {
		cache[endpoint] = null;
	}
	return cache[endpoint];
}

async function pooledMap(items, concurrency, worker) {
	const queue = [...items];
	const workers = Array.from({length: Math.min(concurrency, queue.length)}, async () => {
		while (queue.length) {
			const item = queue.shift();
			await worker(item);
		}
	});
	await Promise.all(workers);
}

const explicitFormLabels = {
	taurospaldeacombat: 'コンバット種',
	taurospaldeablaze: 'ブレイズ種',
	taurospaldeaaqua: 'アクア種',
	greninjabond: 'きずなへんげ',
	mausholdfour: '４ひきかぞく',
	squawkabillyblue: 'ブルーフェザー',
	squawkabillywhite: 'ホワイトフェザー',
	squawkabillyyellow: 'イエローフェザー',
	pichuspikyeared: 'ギザみみ',
	necrozmadawnwings: 'あかつきのつばさ',
	necrozmaduskmane: 'たそがれのたてがみ',
};

const explicitPokeApiFormAliases = {
	basculegionf: 'basculegion-female',
	taurospaldeacombat: 'tauros-paldea-combat-breed',
	taurospaldeablaze: 'tauros-paldea-blaze-breed',
	taurospaldeaaqua: 'tauros-paldea-aqua-breed',
	ogerponwellspring: 'ogerpon-wellspring-mask',
	ogerponhearthflame: 'ogerpon-hearthflame-mask',
	ogerponcornerstone: 'ogerpon-cornerstone-mask',
	greninjabond: 'greninja-battle-bond',
	mausholdfour: 'maushold-family-of-four',
	squawkabillyblue: 'squawkabilly-blue-plumage',
	squawkabillywhite: 'squawkabilly-white-plumage',
	squawkabillyyellow: 'squawkabilly-yellow-plumage',
};

const explicitResourceLabels = {
	moves: {
		gmaxbefuddle: 'キョダイコワク',
		gmaxcannonade: 'キョダイホウゲキ',
		gmaxcentiferno: 'キョダイヒャッカ',
		gmaxchistrike: 'キョダイシンゲキ',
		gmaxcuddle: 'キョダイホーヨー',
		gmaxdepletion: 'キョダイゲンスイ',
		gmaxdrumsolo: 'キョダイコランダ',
		gmaxfinale: 'キョダイダンエン',
		gmaxfireball: 'キョダイカキュウ',
		gmaxfoamburst: 'キョダイホウマツ',
		gmaxgoldrush: 'キョダイコバン',
		gmaxgravitas: 'キョダイテンドウ',
		gmaxhydrosnipe: 'キョダイソゲキ',
		gmaxmalodor: 'キョダイシュウキ',
		gmaxmeltdown: 'キョダイユウゲキ',
		gmaxoneblow: 'キョダイイチゲキ',
		gmaxrapidflow: 'キョダイレンゲキ',
		gmaxreplenish: 'キョダイサイセイ',
		gmaxresonance: 'キョダイセンリツ',
		gmaxsandblast: 'キョダイサジン',
		gmaxsmite: 'キョダイテンバツ',
		gmaxsnooze: 'キョダイスイマ',
		gmaxsteelsurge: 'キョダイコウジン',
		gmaxstonesurge: 'キョダイガンジン',
		gmaxstunshock: 'キョダイカンデン',
		gmaxsweetness: 'キョダイカンロ',
		gmaxtartness: 'キョダイサンゲキ',
		gmaxterror: 'キョダイゲンエイ',
		gmaxvinelash: 'キョダイベンタツ',
		gmaxvolcalith: 'キョダイフンセキ',
		gmaxvoltcrash: 'キョダイバンライ',
		gmaxwildfire: 'キョダイゴクエン',
		gmaxwindrage: 'キョダイフウゲキ',
	},
	items: {
		aloraichiumz: 'アローラライチュウＺ',
		buginiumz: 'ムシＺ',
		darkiniumz: 'アクＺ',
		decidiumz: 'ジュナイパーＺ',
		dragoniumz: 'ドラゴンＺ',
		eeviumz: 'イーブイＺ',
		electriumz: 'デンキＺ',
		fairiumz: 'フェアリーＺ',
		fightiniumz: 'カクトウＺ',
		firiumz: 'ホノオＺ',
		flyiniumz: 'ヒコウＺ',
		ghostiumz: 'ゴーストＺ',
		grassiumz: 'クサＺ',
		groundiumz: 'ジメンＺ',
		iciumz: 'コオリＺ',
		inciniumz: 'ガオガエンＺ',
		kommoniumz: 'ジャラランガＺ',
		leek: 'ながねぎ',
		lunaliumz: 'ルナアーラＺ',
		lycaniumz: 'ルガルガンＺ',
		mail: 'メール',
		marshadiumz: 'マーシャドーＺ',
		steeliumz: 'ハガネＺ',
		mewniumz: 'ミュウＺ',
		wateriumz: 'ミズＺ',
		mimikiumz: 'ミミッキュＺ',
		metalalloy: 'ふくごうきんぞく',
		normaliumz: 'ノーマルＺ',
		pikaniumz: 'ピカチュウＺ',
		pikashuniumz: 'サトピカＺ',
		poisoniumz: 'ドクＺ',
		prettyfeather: 'きれいなはね',
		primariumz: 'アシレーヌＺ',
		psychiumz: 'エスパーＺ',
		rockiumz: 'イワＺ',
		snorliumz: 'カビゴンＺ',
		solganiumz: 'ソルガレオＺ',
		tapuniumz: 'カプＺ',
		ultranecroziumz: 'ウルトラネクロＺ',
		berry: 'きのみ',
		bitterberry: 'にがいきのみ',
		burntberry: 'やけたきのみ',
		goldberry: 'きんのきのみ',
		iceberry: 'こおりきのみ',
		mintberry: 'ミントのみ',
		miracleberry: 'きせきのみ',
		mysteryberry: 'ふしぎのみ',
		pinkbow: 'ピンクのリボン',
		polkadotbow: 'みずたまリボン',
		przcureberry: 'まひなおしのみ',
		psncureberry: 'どくけしのみ',
		berserkgene: 'はかいのいでんし',
		crucibellite: 'クルーシベライト',
		vilevial: 'ヴァイルバイアル',
	},
	abilities: {
		noability: 'なし',
	},
};

const fallbackFormNames = {
	Alola: 'アローラのすがた',
	AlolaTotem: 'アローラのぬし',
	Starter: 'スターター',
	Galar: 'ガラルのすがた',
	Hisui: 'ヒスイのすがた',
	Paldea: 'パルデアのすがた',
	Gmax: 'キョダイマックス',
	Totem: 'ぬし',
	Mega: 'メガ',
	'Mega-X': 'メガX',
	'Mega-Y': 'メガY',
	Primal: 'ゲンシ',
	Origin: 'オリジン',
	'Origin-Therian': 'オリジン',
	Therian: 'れいじゅうフォルム',
	Incarnate: 'けしんフォルム',
	Attack: 'アタックフォルム',
	Defense: 'ディフェンスフォルム',
	Speed: 'スピードフォルム',
	Complete: 'パーフェクトフォルム',
	Complete10: 'パーフェクトフォルム',
	School: 'むれたすがた',
	Busted: 'ばれたすがた',
	BustedTotem: 'ばれたすがた',
	LowKey: 'ローなすがた',
	Amped: 'ハイなすがた',
	Hero: 'マイティフォーム',
	Crowned: 'れきせんのゆうしゃ',
	CrownedShield: 'たてのおう',
	CrownedSword: 'けんのおう',
	RapidStrike: 'れんげきのかた',
	SingleStrike: 'いちげきのかた',
	Combat: 'コンバット種',
	Blaze: 'ブレイズ種',
	Aqua: 'アクア種',
	White: 'ホワイト',
	Black: 'ブラック',
	Resolute: 'かくごのすがた',
	Pirouette: 'ステップフォルム',
	Sky: 'スカイフォルム',
	Land: 'ランドフォルム',
	Plant: 'プラントフォルム',
	Sandy: 'すなちのミノ',
	Trash: 'ゴミのミノ',
	Heat: 'ヒートロトム',
	Wash: 'ウォッシュロトム',
	Frost: 'フロストロトム',
	Fan: 'スピンロトム',
	Mow: 'カットロトム',
	Original: 'オリジナルキャップ',
	Hoenn: 'ホウエンキャップ',
	Sinnoh: 'シンオウキャップ',
	Unova: 'イッシュキャップ',
	Kalos: 'カロスキャップ',
	Partner: 'キミにきめたキャップ',
	World: 'ワールドキャップ',
	Sunshine: 'ポジフォルム',
	Spring: 'はるのすがた',
	Summer: 'なつのすがた',
	Autumn: 'あきのすがた',
	Winter: 'ふゆのすがた',
	Douse: 'アクアカセット',
	Shock: 'イナズマカセット',
	Burn: 'ブレイズカセット',
	Chill: 'フリーズカセット',
	Wellspring: 'いどのめん',
	Hearthflame: 'かまどのめん',
	Cornerstone: 'いしずえのめん',
	Teal: 'みどりのめん',
	Tera: 'テラスタル',
	Terastal: 'テラスタルフォルム',
	Stellar: 'ステラフォルム',
	Female: 'メスのすがた',
	'White-Striped': 'しろすじのすがた',
	'Blue-Striped': 'あおすじのすがた',
	'Red-Striped': 'あかすじのすがた',
	Meteor: 'りゅうせいのすがた',
};

const fallbackFormTokens = {
	alola: 'アローラ',
	alolatotem: 'アローラのぬし',
	galar: 'ガラル',
	hisui: 'ヒスイ',
	paldea: 'パルデア',
	gmax: 'キョダイマックス',
	totem: 'ぬし',
	mega: 'メガ',
	x: 'X',
	y: 'Y',
	primal: 'ゲンシ',
	origin: 'オリジン',
	therian: 'れいじゅう',
	incarnate: 'けしん',
	attack: 'アタック',
	defense: 'ディフェンス',
	speed: 'スピード',
	complete: 'パーフェクト',
	school: 'むれ',
	busted: 'ばれた',
	lowkey: 'ロー',
	amped: 'ハイ',
	hero: 'マイティ',
	crowned: 'れきせん',
	shield: 'たて',
	sword: 'けん',
	rapid: 'れんげき',
	strike: 'かた',
	single: 'いちげき',
	combat: 'コンバット',
	blaze: 'ブレイズ',
	aqua: 'アクア',
	white: 'ホワイト',
	black: 'ブラック',
	resolute: 'かくご',
	pirouette: 'ステップ',
	sky: 'スカイ',
	land: 'ランド',
	plant: 'プラント',
	sandy: 'すなち',
	trash: 'ゴミ',
	heat: 'ヒート',
	wash: 'ウォッシュ',
	frost: 'フロスト',
	fan: 'スピン',
	mow: 'カット',
	original: 'オリジナル',
	hoenn: 'ホウエン',
	sinnoh: 'シンオウ',
	unova: 'イッシュ',
	kalos: 'カロス',
	partner: 'キミにきめた',
	world: 'ワールド',
	sunshine: 'ポジ',
	spring: 'はる',
	summer: 'なつ',
	autumn: 'あき',
	winter: 'ふゆ',
	douse: 'アクア',
	shock: 'イナズマ',
	burn: 'ブレイズ',
	chill: 'フリーズ',
	wellspring: 'いどのめん',
	hearthflame: 'かまどのめん',
	cornerstone: 'いしずえのめん',
	teal: 'みどりのめん',
	tera: 'テラスタル',
	terastal: 'テラスタル',
	stellar: 'ステラ',
	female: 'メス',
	male: 'オス',
	dawn: 'あかつき',
	wings: 'つばさ',
	dusk: 'たそがれ',
	mane: 'たてがみ',
	meteor: 'りゅうせい',
	pop: 'ポップ',
	star: 'スター',
	belle: 'ベル',
	cosplay: 'コスプレ',
	libre: 'ルチャ',
	phd: 'ドクター',
	spiky: 'ギザ',
	eared: 'みみ',
	striped: 'すじ',
	normal: 'ノーマル',
	fighting: 'かくとう',
	flying: 'ひこう',
	poison: 'どく',
	ground: 'じめん',
	rock: 'いわ',
	bug: 'むし',
	ghost: 'ゴースト',
	steel: 'はがね',
	fire: 'ほのお',
	water: 'みず',
	grass: 'くさ',
	electric: 'でんき',
	psychic: 'エスパー',
	ice: 'こおり',
	dragon: 'ドラゴン',
	dark: 'あく',
	fairy: 'フェアリー',
};

function translateFallbackForm(forme) {
	if (!forme) return '';
	if (fallbackFormNames[forme]) return fallbackFormNames[forme];
	const compact = forme.replace(/[^A-Za-z0-9]+/g, '');
	if (fallbackFormNames[compact]) return fallbackFormNames[compact];
	const translated = forme.split('-').map(token => fallbackFormTokens[token.toLowerCase()] || token);
	return translated.join('・');
}

function addFormLabelVariants(target, identifier, label) {
	const raw = `${identifier || ''}`.toLowerCase();
	if (!raw || !label) return;
	const variants = new Set([canonicalId(raw)]);
	for (const suffix of ['-mask', '-breed']) {
		if (raw.endsWith(suffix)) variants.add(canonicalId(raw.slice(0, -suffix.length)));
	}
	if (raw.endsWith('-female')) variants.add(canonicalId(raw.replace(/-female$/, '-f')));
	if (raw.endsWith('-male')) variants.add(canonicalId(raw.replace(/-male$/, '')));
	for (const variant of variants) {
		if (!target[variant]) target[variant] = label;
	}
}

function combineBaseAndForm(baseJapanese, formLabel, entry) {
	if (!formLabel) return baseJapanese || entry.name;
	if (!entry.forme) return baseJapanese || entry.name;
	if (!baseJapanese) return `${entry.baseSpecies || entry.name} (${formLabel})`;
	if (formLabel.includes(baseJapanese)) return formLabel;
	return `${baseJapanese}（${formLabel}）`;
}

function makeFallbackSpeciesName(entry, baseJapanese, formLabelMap) {
	if (!entry.forme) return baseJapanese || entry.name;
	const canonicalName = canonicalId(entry.name);
	const formLabel = explicitFormLabels[canonicalName] || formLabelMap[canonicalName] || translateFallbackForm(entry.forme);
	return combineBaseAndForm(baseJapanese, formLabel, entry);
}

function isFetchableOfficialEntry(entry) {
	return !!(entry && Number(entry.num) > 0 && entry.isNonstandard !== 'CAP' && entry.isNonstandard !== 'Custom');
}

function makeApiSlug(text) {
	return `${text || ''}`
		.normalize('NFKD')
		.toLowerCase()
		.replace(/[’']/g, '')
		.replace(/♀/g, '-female')
		.replace(/♂/g, '-male')
		.replace(/[^a-z0-9]+/g, '-')
		.replace(/^-+|-+$/g, '');
}

async function supplementBaseSpeciesNames(Pokedex, speciesNames, cache) {
	const missingNums = [...new Set(Object.values(Pokedex)
		.filter(isFetchableOfficialEntry)
		.map(entry => String(entry.num))
		.filter(num => isEnglishish(speciesNames[num])))
	];
	await pooledMap(missingNums, 8, async num => {
		const name = await fetchCachedPokeApiName(`pokemon-species/${num}`, data => extractLocalizedName(data.names), cache);
		if (name) speciesNames[num] = name;
	});
}

async function supplementResourceLabelsById(resources, endpoint, labelsById, cache, explicitLabelsById = {}) {
	const missingIds = Object.entries(resources)
		.filter(([id, resource]) => explicitLabelsById[id] || Number(resource.num) > 0)
		.map(([id]) => id)
		.filter(id => isEnglishish(labelsById[id]));
	await pooledMap(missingIds, 8, async id => {
		if (explicitLabelsById[id]) {
			labelsById[id] = explicitLabelsById[id];
			return;
		}
		const resource = resources[id];
		const slug = makeApiSlug(resource.name);
		let name = slug
			? await fetchCachedPokeApiName(`${endpoint}/${slug}`, data => extractLocalizedName(data.names), cache)
			: null;
		if (!name && endpoint !== 'item' && Number(resource.num) > 0) {
			name = await fetchCachedPokeApiName(`${endpoint}/${resource.num}`, data => extractLocalizedName(data.names), cache);
		}
		if (name) labelsById[id] = name;
	});
}

function buildFormLabelMap(pokemonFormsRows, pokemonFormNamesRows) {
	const formNamesById = {};
	for (const row of pokemonFormNamesRows) {
		const priority = preferredLanguageIds.indexOf(row.local_language_id);
		if (priority < 0 || !row.form_name) continue;
		const current = formNamesById[row.pokemon_form_id];
		if (!current || priority < current.priority) {
			formNamesById[row.pokemon_form_id] = {name: row.form_name, priority};
		}
	}
	const map = {};
	for (const row of pokemonFormsRows) {
		const formName = formNamesById[row.id]?.name;
		if (!formName) continue;
		addFormLabelVariants(map, row.identifier, formName);
	}
	for (const [canonical, label] of Object.entries(explicitFormLabels)) {
		map[canonical] = label;
	}
	return map;
}

function getPokeApiFormIdentifierCandidates(entry) {
	const canonicalName = canonicalId(entry.name);
	const candidates = [];
	if (explicitPokeApiFormAliases[canonicalName]) candidates.push(explicitPokeApiFormAliases[canonicalName]);
	candidates.push(makeApiSlug(entry.name));
	return [...new Set(candidates.filter(Boolean))];
}

async function supplementFormLabels(Pokedex, formLabelMap, cache) {
	const missingForms = Object.entries(Pokedex)
		.filter(([, entry]) => isFetchableOfficialEntry(entry) && entry.forme && !formLabelMap[canonicalId(entry.name)]);
	await pooledMap(missingForms, 6, async ([, entry]) => {
		for (const candidate of getPokeApiFormIdentifierCandidates(entry)) {
			const label = await fetchCachedPokeApiName(
				`pokemon-form/${candidate}`,
				data => extractLocalizedName(data.form_names) || extractLocalizedName(data.names),
				cache
			);
			if (!label) continue;
			addFormLabelVariants(formLabelMap, candidate, label);
			addFormLabelVariants(formLabelMap, entry.name, label);
			break;
		}
	});
}

function getBaseSpeciesJapanese(entry, Pokedex, speciesNames) {
	const direct = speciesNames[String(entry.num)] || '';
	if (direct) return direct;
	const baseId = canonicalId(entry.baseSpecies || entry.name);
	const baseEntry = Pokedex[baseId];
	if (!baseEntry) return '';
	return speciesNames[String(baseEntry.num)] || '';
}

function buildSpeciesMap(Pokedex, speciesNames, formLabelMap) {
	const speciesMap = {};
	for (const id of Object.keys(Pokedex).sort()) {
		const entry = Pokedex[id];
		const baseJapanese = getBaseSpeciesJapanese(entry, Pokedex, speciesNames);
		speciesMap[id] = makeFallbackSpeciesName(entry, baseJapanese, formLabelMap);
	}
	return speciesMap;
}

function getItemSpeciesLabel(speciesName, Pokedex, speciesMap) {
	if (!speciesName) return '';
	let speciesId = canonicalId(speciesName);
	let entry = Pokedex[speciesId];
	if (entry?.baseSpecies) {
		speciesId = canonicalId(entry.baseSpecies);
		entry = Pokedex[speciesId];
	}
	return speciesMap[speciesId] || '';
}

function getMegaStoneSuffix(item) {
	if (/\s+X$/i.test(item.name)) return 'ナイトＸ';
	if (/\s+Y$/i.test(item.name)) return 'ナイトＹ';
	if (/\s+Z$/i.test(item.name) || Object.values(item.megaStone || {}).some(name => /-Mega-Z$/i.test(name))) {
		return 'ナイトＺ';
	}
	return 'ナイト';
}

function supplementItemLabelsFromRules(Items, itemLabelsById, Pokedex, speciesMap) {
	for (const [id, item] of Object.entries(Items)) {
		if (!isEnglishish(itemLabelsById[id])) continue;
		if (explicitResourceLabels.items[id]) {
			itemLabelsById[id] = explicitResourceLabels.items[id];
			continue;
		}
		if (item.zMoveType) {
			const typeLabel = fallbackFormTokens[item.zMoveType.toLowerCase()] || item.zMoveType;
			itemLabelsById[id] = `${typeLabel}Ｚ`;
			continue;
		}
		if (item.zMove && item.itemUser?.length) {
			const speciesLabel = getItemSpeciesLabel(item.itemUser[0], Pokedex, speciesMap);
			if (speciesLabel && !isEnglishish(speciesLabel)) {
				itemLabelsById[id] = `${speciesLabel}Ｚ`;
				continue;
			}
		}
		if (item.megaStone) {
			const megaUsers = item.itemUser?.length ? item.itemUser : Object.keys(item.megaStone);
			const speciesLabel = getItemSpeciesLabel(megaUsers[0], Pokedex, speciesMap);
			if (speciesLabel && !isEnglishish(speciesLabel)) {
				itemLabelsById[id] = `${speciesLabel}${getMegaStoneSuffix(item)}`;
				continue;
			}
		}
	}
}

function writeJS(filename, value) {
	fs.mkdirSync(dataDir, {recursive: true});
	fs.writeFileSync(filename, `exports.BattleJapaneseNames = ${JSON.stringify(value)};\n`);
}

async function main() {
	const Pokedex = require(showdownDistPath(showdownRoot, 'data/pokedex.js')).Pokedex;
	const Moves = require(showdownDistPath(showdownRoot, 'data/moves.js')).Moves;
	const Items = require(showdownDistPath(showdownRoot, 'data/items.js')).Items;
	const Abilities = require(showdownDistPath(showdownRoot, 'data/abilities.js')).Abilities;

	const [
		pokemonSpeciesNamesRows,
		moveNamesRows,
		abilityNamesRows,
		typeNamesRows,
		pokemonFormsRows,
		pokemonFormNamesRows,
	] = await Promise.all([
		fetchCSV('pokemon_species_names.csv'),
		fetchCSV('move_names.csv'),
		fetchCSV('ability_names.csv'),
		fetchCSV('type_names.csv'),
		fetchCSV('pokemon_forms.csv'),
		fetchCSV('pokemon_form_names.csv'),
	]);

	const speciesNames = mapNamesById(pokemonSpeciesNamesRows, 'pokemon_species_id');
	const moveNames = mapNamesById(moveNamesRows, 'move_id');
	const abilityNames = mapNamesById(abilityNamesRows, 'ability_id');
	const typeNamesByNumericId = mapNamesById(typeNamesRows, 'type_id');
	const formLabelMap = buildFormLabelMap(pokemonFormsRows, pokemonFormNamesRows);
	const cache = loadCache();
	const moveLabelsById = {};
	const itemLabelsById = {};
	const abilityLabelsById = {};

	for (const [id, move] of Object.entries(Moves)) {
		moveLabelsById[id] = moveNames[String(move.num)] || move.name;
	}
	for (const [id, item] of Object.entries(Items)) {
		itemLabelsById[id] = item.name;
	}
	for (const [id, ability] of Object.entries(Abilities)) {
		abilityLabelsById[id] = abilityNames[String(ability.num)] || ability.name;
	}

	await supplementBaseSpeciesNames(Pokedex, speciesNames, cache);
	await supplementResourceLabelsById(Moves, 'move', moveLabelsById, cache, explicitResourceLabels.moves);
	await supplementResourceLabelsById(Items, 'item', itemLabelsById, cache, explicitResourceLabels.items);
	await supplementResourceLabelsById(Abilities, 'ability', abilityLabelsById, cache, explicitResourceLabels.abilities);
	await supplementFormLabels(Pokedex, formLabelMap, cache);
	saveCache(cache);
	const speciesMap = buildSpeciesMap(Pokedex, speciesNames, formLabelMap);
	supplementItemLabelsFromRules(Items, itemLabelsById, Pokedex, speciesMap);

	const payload = {
		species: speciesMap,
		moves: moveLabelsById,
		items: itemLabelsById,
		abilities: abilityLabelsById,
		types: {
			normal: typeNamesByNumericId['1'] || 'ノーマル',
			fighting: typeNamesByNumericId['2'] || 'かくとう',
			flying: typeNamesByNumericId['3'] || 'ひこう',
			poison: typeNamesByNumericId['4'] || 'どく',
			ground: typeNamesByNumericId['5'] || 'じめん',
			rock: typeNamesByNumericId['6'] || 'いわ',
			bug: typeNamesByNumericId['7'] || 'むし',
			ghost: typeNamesByNumericId['8'] || 'ゴースト',
			steel: typeNamesByNumericId['9'] || 'はがね',
			fire: typeNamesByNumericId['10'] || 'ほのお',
			water: typeNamesByNumericId['11'] || 'みず',
			grass: typeNamesByNumericId['12'] || 'くさ',
			electric: typeNamesByNumericId['13'] || 'でんき',
			psychic: typeNamesByNumericId['14'] || 'エスパー',
			ice: typeNamesByNumericId['15'] || 'こおり',
			dragon: typeNamesByNumericId['16'] || 'ドラゴン',
			dark: typeNamesByNumericId['17'] || 'あく',
			fairy: typeNamesByNumericId['18'] || 'フェアリー',
			stellar: 'ステラ',
		},
		categories: {
			physical: 'ぶつり',
			special: 'とくしゅ',
			status: 'へんか',
		},
	};

	writeJS(outputPath, payload);
	console.log(`Built ${path.relative(rootDir, outputPath)}`);
}

main().catch(error => {
	console.error(error);
	process.exit(1);
});
