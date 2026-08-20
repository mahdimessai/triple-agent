import test from "node:test";
import assert from "node:assert/strict";
import { existsSync, readFileSync, readdirSync, statSync } from "node:fs";
import { dirname, extname, join, relative, resolve } from "node:path";

const root = process.cwd();
const app = join(root, "app");
const features = join(root, "features");
const game = join(features, "triple-agent");
const components = join(root, "components");

function sourceFiles(directory) {
  if (!existsSync(directory)) return [];
  const result = [];
  for (const name of readdirSync(directory)) {
    const path = join(directory, name);
    if (statSync(path).isDirectory()) result.push(...sourceFiles(path));
    else if ([".ts", ".tsx", ".mjs"].includes(extname(path))) result.push(path);
  }
  return result;
}

function read(path) { return readFileSync(path, "utf8"); }
function projectFiles() { return [...sourceFiles(app), ...sourceFiles(features), ...sourceFiles(components)]; }

const gameFiles = sourceFiles(game);
const gameSource = gameFiles.map(read).join("\n");

test("Next.js app is routing and composition, not the Triple Agent implementation", () => {
  assert.equal(existsSync(join(app, "_game")), false, "game implementation must not live under app/");
  assert.equal(existsSync(game), true, "Triple Agent must have an explicit feature home");
  const page = read(join(app, "page.tsx"));
  assert.match(page, /@\/features\/triple-agent\/game/);
});

test("feature code never depends on the routing layer", () => {
  for (const file of gameFiles) {
    const source = read(file);
    assert.equal(source.includes("@/app/"), false, `${relative(root, file)} imports app/`);
    assert.equal(source.includes("../../app/"), false, `${relative(root, file)} reaches into app/`);
  }
});

test("shared UI cannot depend on Triple Agent feature internals", () => {
  for (const file of sourceFiles(components)) {
    const source = read(file);
    assert.equal(source.includes("@/features/triple-agent"), false, `${relative(root, file)} imports Triple Agent`);
    assert.equal(source.includes("../features/triple-agent"), false, `${relative(root, file)} imports Triple Agent`);
  }
});

test("production game code never imports the mock workbench", () => {
  assert.equal(gameSource.includes("/mock"), false);
  assert.equal(gameSource.includes("../mock"), false);
});

test("Game handles every protocol phase exactly once or in an intentional group", () => {
  const protocol = read(join(game, "protocol.ts"));
  const gameRoot = read(join(game, "game.tsx"));
  const phaseBlock = protocol.slice(protocol.indexOf("export type Phase"), protocol.indexOf("export type Faction"));
  const phases = new Set([...phaseBlock.matchAll(/"([A-Z_]+)"/g)].map((match) => match[1]));
  const covered = new Set([...gameRoot.matchAll(/case "([A-Z_]+)"/g)].map((match) => match[1]));
  assert.deepEqual([...covered].sort(), [...phases].sort());
  assert.match(gameRoot, /assertNever\(phase\)/);
});

test("screen command literals are members of ClientCommand", () => {
  const protocol = read(join(game, "protocol.ts"));
  const commandBlock = protocol.slice(protocol.indexOf("export type ClientCommand"), protocol.indexOf("export type ResyncCommand"));
  const valid = new Set([...commandBlock.matchAll(/\{\s*kind:\s*"([^"]+)"/g)].map((match) => match[1]));
  const screens = sourceFiles(join(game, "screens")).map(read).join("\n");
  const emitted = new Set([...screens.matchAll(/kind:\s*"([^"]+)"/g)].map((match) => match[1]));
  for (const kind of emitted) assert.equal(valid.has(kind), true, `screen emits unknown command ${kind}`);
});

test("all relative source imports resolve across app, features, and shared components", () => {
  const files = projectFiles();
  const known = new Set(files.map((path) => resolve(path)));
  for (const file of files) {
    for (const match of read(file).matchAll(/(?:from\s+|import\s*\()(["'])([^"']+)\1/g)) {
      const specifier = match[2];
      if (!specifier.startsWith(".")) continue;
      const base = resolve(dirname(file), specifier);
      const candidates = [base, `${base}.ts`, `${base}.tsx`, `${base}.mjs`, join(base, "index.ts"), join(base, "index.tsx")];
      assert.equal(candidates.some((candidate) => known.has(resolve(candidate))), true, `${relative(root, file)} has unresolved import ${specifier}`);
    }
  }
});
