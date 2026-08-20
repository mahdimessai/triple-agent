import path from "node:path";
import type { NextConfig } from "next";

const basePageExtensions = ["tsx", "ts", "jsx", "js"];

const nextConfig: NextConfig = {
  reactStrictMode: true,
  output: "standalone",
  pageExtensions: process.env.NODE_ENV === "development" ? [...basePageExtensions, "dev.tsx"] : basePageExtensions,
  turbopack: {
    root: path.resolve(__dirname),
  },
};

export default nextConfig;
