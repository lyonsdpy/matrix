import path from "node:path";
import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // 显式锁定工作区根为本目录，避免 Next 向上撞到家目录游离的 pnpm-lock 误判 root
  turbopack: {
    root: path.join(__dirname),
  },
};

export default nextConfig;
