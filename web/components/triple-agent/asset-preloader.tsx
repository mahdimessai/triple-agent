"use client";

import { useEffect } from "react";
import { art } from "./asset-registry";

export function AssetPreloader() {
  useEffect(() => {
    // Pre-cache and pre-decode all game art assets up front
    Object.values(art).forEach((item) => {
      const img = new window.Image();
      img.src = item.src;
      if ("decode" in img) {
        img.decode().catch(() => {
          // Ignore decode errors for non-blocking preloading
        });
      }
    });
  }, []);

  return null;
}
