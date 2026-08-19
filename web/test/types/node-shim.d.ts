/* Self-contained declarations used only by test/tsconfig.json. The main app
   typecheck excludes this directory and uses @types/node from devDependencies. */
declare const process: { env: Record<string, string | undefined> };

declare module "node:test" {
  type TestFn = (name: string, fn: () => void | Promise<void>) => void;
  const test: TestFn;
  export default test;
}

declare module "node:assert/strict" {
  const assert: {
    equal(actual: unknown, expected: unknown, message?: string): void;
    deepEqual(actual: unknown, expected: unknown, message?: string): void;
    match(actual: string, expected: RegExp, message?: string): void;
  };
  export default assert;
}
