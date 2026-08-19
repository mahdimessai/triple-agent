import path from "node:path";
import type { NextConfig } from "next";

const repoRoot = path.resolve(process.cwd(), "..");
const basePageExtensions = ["tsx", "ts", "jsx", "js"];

const nextConfig: NextConfig = {
  reactStrictMode: true,
  output: "standalone",
  outputFileTracingRoot: repoRoot,
  pageExtensions: process.env.NODE_ENV === "development" ? [...basePageExtensions, "dev.tsx"] : basePageExtensions,
  turbopack: { root: repoRoot },
};

export default nextConfig;
