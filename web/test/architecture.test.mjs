import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync, readdirSync, statSync } from "node:fs";
import { extname, join, relative } from "node:path";

const root = process.cwd();
const app = join(root, "app");
const feature = join(root, "features", "triple-agent");

function sourceFiles(directory) {
  const result = [];
  for (const name of readdirSync(directory)) {
    const path = join(directory, name);
    if (statSync(path).isDirectory()) result.push(...sourceFiles(path));
    else if ([".ts", ".tsx", ".mjs"].includes(extname(path))) result.push(path);
  }
  return result;
}

function importsOf(path) {
  const source = readFileSync(path, "utf8");
  return [...source.matchAll(/(?:from\s+|import\s*\()(["'])([^"']+)\1/g)].map((match) => match[2]);
}

function assertNoImport(files, forbidden, message) {
  for (const file of files) {
    for (const specifier of importsOf(file)) {
      assert.equal(forbidden.some((pattern) => pattern.test(specifier)), false, `${relative(root, file)} ${message}: ${specifier}`);
    }
  }
}

test("app routes compose features rather than reaching into realtime internals", () => {
  assertNoImport(sourceFiles(app), [
    /features\/triple-agent\/(?:protocol|transport|session)(?:\/|$)/,
    /app\/_game/,
  ], "must not import Triple Agent internals");
});

test("protocol is a pure trust boundary", () => {
  assertNoImport(sourceFiles(join(feature, "protocol")), [
    /^react(?:\/|$)/,
    /(?:^|\/)transport(?:\/|$)/,
    /(?:^|\/)session(?:\/|$)/,
    /(?:^|\/)screens(?:\/|$)/,
  ], "violates the protocol boundary");
});

test("transport has no React or presentation dependencies", () => {
  assertNoImport(sourceFiles(join(feature, "transport")), [
    /^react(?:\/|$)/,
    /(?:^|\/)screens(?:\/|$)/,
    /(?:^|\/)session(?:\/|$)/,
  ], "violates the transport boundary");
});

test("session lifecycle cannot depend on screens", () => {
  assertNoImport(sourceFiles(join(feature, "session")), [/(?:^|\/)screens(?:\/|$)/], "violates the session boundary");
});

test("feature public entry stays deliberately small", () => {
  const source = readFileSync(join(feature, "index.ts"), "utf8");
  assert.match(source, /TripleAgentGame/);
  assert.equal(source.includes("export *"), false, "feature index must not become a barrel dump");
});
