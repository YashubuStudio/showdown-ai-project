"use strict";

const fs = require("fs");
const path = require("path");
const child_process = require("child_process");

const clientRoot = path.resolve(__dirname, "..");
const localShowdownRoot = path.resolve(clientRoot, "../pokemon-showdown-local");
const cacheShowdownRoot = path.resolve(clientRoot, "caches/pokemon-showdown");

function fileExists(root, relativePath) {
	return fs.existsSync(path.join(root, relativePath));
}

function hasUsableLocalShowdown() {
	return (
		fs.existsSync(localShowdownRoot) &&
		fileExists(localShowdownRoot, "package.json") &&
		fileExists(localShowdownRoot, "dist/sim/dex.js") &&
		fileExists(localShowdownRoot, "dist/server/chat.js")
	);
}

function ensureCacheCheckout() {
	if (!fs.existsSync(cacheShowdownRoot)) {
		child_process.execSync("git clone https://github.com/smogon/pokemon-showdown.git", {
			cwd: path.join(clientRoot, "caches"),
			stdio: "inherit",
		});
	}
}

function ensureBuiltShowdown(root) {
	if (!fileExists(root, "dist/sim/dex.js") || !fileExists(root, "dist/server/chat.js")) {
		child_process.execSync("npm run build", {
			cwd: root,
			stdio: "inherit",
		});
	}
}

function resolveShowdownRoot() {
	if (hasUsableLocalShowdown()) return localShowdownRoot;
	ensureCacheCheckout();
	ensureBuiltShowdown(cacheShowdownRoot);
	return cacheShowdownRoot;
}

function showdownDistPath(root, relativePath) {
	return path.join(root, "dist", relativePath);
}

function showdownSourcePath(root, relativePath) {
	return path.join(root, relativePath);
}

module.exports = {
	resolveShowdownRoot,
	showdownDistPath,
	showdownSourcePath,
};
