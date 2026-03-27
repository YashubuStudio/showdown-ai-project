'use strict';

const port = Number(process.env.PS_PORT || 8000);
const bindaddress = process.env.PS_BIND_ADDRESS || '0.0.0.0';
const clientPort = Number(process.env.PS_CLIENT_PORT || 8080);
const serverid = process.env.PS_SERVER_ID || 'koharulocal';
const lanProfile = process.env.PS_LAN_PROFILE || 'lightweight';
const isLightweightProfile = lanProfile !== 'training';

function escapeHtml(str) {
	return String(str)
		.replace(/&/g, '&amp;')
		.replace(/</g, '&lt;')
		.replace(/>/g, '&gt;')
		.replace(/"/g, '&quot;');
}

function getHostname(req) {
	const hostHeader = req.headers.host || `localhost:${port}`;
	return hostHeader.replace(/:\d+$/, '');
}

exports.port = port;
exports.bindaddress = bindaddress;
exports.serverid = serverid;

// Local LAN setup: allow quick usernames without the public login server.
exports.noguestsecurity = true;
exports.nobattlesearch = true;
exports.repl = false;
exports.crashguard = false;
exports.loglevel = 5;
exports.nothrottle = isLightweightProfile;
exports.noipchecks = isLightweightProfile;
exports.subprocesses = isLightweightProfile ? 0 : {
	network: 1,
	simulator: 1,
	validator: 1,
	verifier: 0,
	localartemis: 0,
	remoteartemis: 0,
	friends: 0,
	chatdb: 0,
	modlog: 0,
	pm: 0,
	battlesearch: 0,
	datasearch: 0,
};

exports.reportjoins = false;
exports.reportjoinsperiod = 0;
exports.reportbattles = false;
exports.reportbattlejoins = false;

// Lightweight LAN mode: disable optional SQLite-backed features and keep the
// server focused on live battles instead of persistence or moderation tooling.
exports.usesqlite = false;
exports.usesqlitemodlog = false;
exports.usesqlitefriends = false;
exports.usesqlitepms = false;

exports.customhttpresponse = function (req, res) {
	if (!req.url || req.url === '/' || req.url === '/index.html') {
		const host = escapeHtml(getHostname(req));
		const safeServerid = escapeHtml(serverid);
		const encodedServerid = encodeURIComponent(serverid);
		const clientUrl =
			`http://${host}:${clientPort}/play.pokemonshowdown.com/lan.html?~~${host}:${port}&serverid=${encodedServerid}`;

		res.writeHead(200, {'Content-Type': 'text/html; charset=utf-8'});
		res.end(`<!doctype html>
<html lang="ja">
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<title>Pokemon Showdown LAN</title>
<style>
body {
	margin: 0;
	font-family: sans-serif;
	background: #f4efe6;
	color: #1f1a17;
}
main {
	max-width: 720px;
	margin: 0 auto;
	padding: 48px 24px;
}
a {
	color: #0044aa;
}
.card {
	background: #fffdf8;
	border: 1px solid #d8cfc0;
	border-radius: 16px;
	padding: 24px;
	box-shadow: 0 12px 30px rgba(0, 0, 0, 0.08);
}
</style>
</head>
<body>
<main>
<div class="card">
<h1>Pokemon Showdown LAN</h1>
<p>この PC または同じローカルネットワーク上の別 PC からローカルクライアントを開いてください。</p>
<p><a href="${clientUrl}">${clientUrl}</a></p>
<p>Server ID: <code>${safeServerid}</code></p>
<p>この構成では公開 Pokémon Showdown ではなく、この LAN サーバーだけに接続します。</p>
</div>
</main>
</body>
</html>`);
		return true;
	}
	return false;
};
