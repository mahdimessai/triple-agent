import test from "node:test";
import assert from "node:assert/strict";
import { existsSync, readFileSync, readdirSync, statSync } from "node:fs";
import { dirname, extname, join, resolve } from "node:path";

const root = process.cwd();
const app = join(root, "app");
const game = join(app, "_game");

function sourceFiles(directory) {
  const result = [];
  for (const name of readdirSync(directory)) {
    const path = join(directory, name);
    if (statSync(path).isDirectory()) result.push(...sourceFiles(path));
    else if ([".ts", ".tsx", ".mjs"].includes(extname(path))) result.push(path);
  }
  return result;
}

function read(path) { return readFileSync(path, "utf8"); }

const gameFiles = sourceFiles(game);
const gameSource = gameFiles.map(read).join("\n");

test("deep-cut target has no legacy architecture directories", () => {
  for (const directory of ["components", "features", "api", "connection", "session", "model", "protocol", "operations"]) {
    assert.equal(existsSync(join(root, directory)), false, `${directory}/ must not exist at web root`);
    assert.equal(existsSync(join(game, directory)), false, `${directory}/ must not exist inside app/_game`);
  }
});

test("transitional screen and command abstractions cannot creep back in", () => {
  for (const token of ["ScreenId", "CommandPayload", "clientCommandFromLegacy", "useRoomSession", "MissionScreen", "next/dynamic", "setScreen("]) {
    assert.equal(gameSource.includes(token), false, `forbidden transitional token: ${token}`);
  }
  assert.equal(/\bclass\s+[A-Za-z_$]/.test(gameSource), false, "frontend application classes are forbidden");
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

test("all relative source imports resolve", () => {
  const files = sourceFiles(app);
  const known = new Set(files.map((path) => resolve(path)));
  for (const file of files) {
    for (const match of read(file).matchAll(/(?:from\s+|import\s*\()(["'])([^"']+)\1/g)) {
      const specifier = match[2];
      if (!specifier.startsWith(".")) continue;
      const base = resolve(dirname(file), specifier);
      const candidates = [base, `${base}.ts`, `${base}.tsx`, `${base}.mjs`, join(base, "index.ts"), join(base, "index.tsx")];
      assert.equal(candidates.some((candidate) => known.has(resolve(candidate))), true, `${file} has unresolved import ${specifier}`);
    }
  }
});
