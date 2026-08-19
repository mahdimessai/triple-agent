import type { Metadata, Viewport } from "next";
import { RetireLegacyServiceWorker } from "./legacy-service-worker";
import "./globals.css";

export const metadata: Metadata = {
  title: "Triple Agent",
  description: "A connected social-deduction game of hidden agencies.",
  applicationName: "Triple Agent",
  appleWebApp: { capable: true, title: "Triple Agent", statusBarStyle: "black-translucent" },
};

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  viewportFit: "cover",
  themeColor: "#a74618",
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body>
        <RetireLegacyServiceWorker />
        {children}
      </body>
    </html>
  );
}
