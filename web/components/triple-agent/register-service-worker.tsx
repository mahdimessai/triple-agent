"use client";

import { useEffect } from "react";

export function RetireLegacyServiceWorker() {
  useEffect(() => {
    if (!("serviceWorker" in navigator)) return;

    void navigator.serviceWorker.getRegistrations().then(async (registrations) => {
      const legacyRegistrations = registrations.filter((registration) => {
        const scriptUrl = registration.active?.scriptURL ?? registration.waiting?.scriptURL ?? registration.installing?.scriptURL;
        return scriptUrl ? new URL(scriptUrl).pathname === "/sw.js" : false;
      });

      if (legacyRegistrations.length === 0) return;

      await Promise.all(legacyRegistrations.map((registration) => registration.unregister()));

      if ("caches" in window) {
        await window.caches.delete("triple-agent-shell-v2");
      }
    });
  }, []);

  return null;
}
