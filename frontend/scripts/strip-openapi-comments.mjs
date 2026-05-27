import { readFileSync, writeFileSync } from "node:fs";

const path = new URL("../types/openapi.ts", import.meta.url);
const source = readFileSync(path, "utf8");
const withoutGeneratedComments = source.replace(/^\s*\/\*\*[\s\S]*?\*\/\n/gm, "");

writeFileSync(path, withoutGeneratedComments);
