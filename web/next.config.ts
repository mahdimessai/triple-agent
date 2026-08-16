import path from "node:path";
import type { NextConfig } from "next";

// The lockfile and node_modules live at the repo root (npm workspace), so file
// tracing has to start there or the standalone build misses hoisted packages.
const repoRoot = path.resolve(process.cwd(), "..");
const basePageExtensions = ["tsx", "ts", "jsx", "js"];

const nextConfig: NextConfig = {
  reactStrictMode: true,
  output: "standalone",
  outputFileTracingRoot: repoRoot,
  // Development-only pages use the .dev.tsx suffix and are omitted from production builds.
  pageExtensions: process.env.NODE_ENV === "development" ? [...basePageExtensions, "dev.tsx"] : basePageExtensions,
  turbopack: {
    root: repoRoot,
  },
};

export default nextConfig;
