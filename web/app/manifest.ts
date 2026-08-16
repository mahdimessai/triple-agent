import type { MetadataRoute } from "next";

export default function manifest(): MetadataRoute.Manifest {
  return {
    name: "Triple Agent",
    short_name: "Triple Agent",
    id: "/",
    description: "A connected social-deduction game of hidden agencies.",
    start_url: "/",
    scope: "/",
    display: "standalone",
    background_color: "#17120f",
    theme_color: "#a74618",
    orientation: "portrait-primary",
    icons: [
      {
        src: "/icon-192.png",
        sizes: "192x192",
        type: "image/png",
      },
      {
        src: "/icon-512.png",
        sizes: "512x512",
        type: "image/png",
      },
    ],
  };
}
