import path from "node:path";
import type { NextConfig } from "next";

// The lockfile and node_modules live at the repo root (npm workspace), so file
// tracing has to start there or the standalone build misses hoisted packages.
const repoRoot = path.resolve(process.cwd(), "..");

const nextConfig: NextConfig = {
  reactStrictMode: true,
  output: "standalone",
  outputFileTracingRoot: repoRoot,
  turbopack: {
    root: repoRoot,
  },
};

export default nextConfig;
