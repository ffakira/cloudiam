import { readdir, readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { join, relative } from "node:path";

const root = fileURLToPath(new URL("../src", import.meta.url));

const forbiddenPatterns = [
	{
		name: "browser persistent storage",
		pattern: /\b(?:localStorage|sessionStorage|indexedDB)\b/,
	},
	{
		name: "document cookie access",
		pattern: /\bdocument\.cookie\b/,
	},
	{
		name: "zustand persistence middleware",
		pattern: /\bfrom\s+["']zustand\/middleware["']|\bpersist\s*\(/,
	},
	{
		name: "client bearer token plumbing",
		pattern: /\bAuthorization\b|\bBearer\b/,
	},
	{
		name: "CloudIAM or console token field",
		pattern: /\b(?:access_token|cloudiam_session_token|console_jwt|jwt|session_token|session_token_hash)\b/i,
	},
	{
		name: "raw device-code field",
		pattern: /\bdevice_code(?:_hash)?\b/i,
	},
	{
		name: "AWS credential field",
		pattern: /\b(?:aws_access_key_id|aws_secret_access_key|aws_session_token|secret_access_key)\b/i,
	},
	{
		name: "raw idempotency key state field",
		pattern: /\bidempotency_key\b/i,
	},
];

const violations = [];

for (const file of await sourceFiles(root)) {
	const body = await readFile(file, "utf8");
	for (const { name, pattern } of forbiddenPatterns) {
		if (pattern.test(body)) {
			violations.push(`${relative(root, file)}: ${name}`);
		}
	}
}

if (violations.length > 0) {
	console.error("Browser secret boundary check failed:");
	for (const violation of violations) {
		console.error(`- ${violation}`);
	}
	process.exit(1);
}

async function sourceFiles(dir) {
	const entries = await readdir(dir, { withFileTypes: true });
	const files = [];

	for (const entry of entries) {
		const path = join(dir, entry.name);
		if (entry.isDirectory()) {
			files.push(...(await sourceFiles(path)));
			continue;
		}
		if (/\.(ts|tsx)$/.test(entry.name)) {
			files.push(path);
		}
	}

	return files;
}
