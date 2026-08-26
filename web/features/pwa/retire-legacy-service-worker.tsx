"use client";

import { useEffect } from "react";

/**
 * Temporary migration cleanup for the retired /sw.js shell cache.
 * Delete this component once clients from the pre-refactor PWA generation are no longer supported.
 */
export function RetireLegacyServiceWorker() {
  useEffect(() => {
    if (!("serviceWorker" in navigator)) return;

    void navigator.serviceWorker.getRegistrations().then(async (registrations) => {
      const legacy = registrations.filter((registration) => {
        const scriptUrl = registration.active?.scriptURL ?? registration.waiting?.scriptURL ?? registration.installing?.scriptURL;
        return scriptUrl ? new URL(scriptUrl).pathname === "/sw.js" : false;
      });
      if (legacy.length === 0) return;
      await Promise.all(legacy.map((registration) => registration.unregister()));
      if ("caches" in window) await window.caches.delete("triple-agent-shell-v2");
    });
  }, []);
  return null;
}
