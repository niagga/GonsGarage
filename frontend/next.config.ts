import type { NextConfig } from "next";
import path from "node:path";
import { fileURLToPath } from "node:url";

// Turbopack (Next 16+) can infer the wrong workspace root (e.g. `src/app`) and then
// fail to resolve `next/package.json`. Pin the root to this app directory.
const projectRoot = path.dirname(fileURLToPath(import.meta.url));

// Standalone solo en imagen Docker (Linux); en Windows `pnpm build` sin esto evita EPERM en symlinks.
const nextConfig: NextConfig = {
  ...(process.env.DOCKER_BUILD === "1" ? { output: "standalone" as const } : {}),
  turbopack: {
    root: projectRoot,
  },
};

export default nextConfig;
