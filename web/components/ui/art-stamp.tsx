import Image from "next/image";
import { art, type ArtName } from "@/components/triple-agent/asset-registry";

export function ArtStamp({ artName, alt, className = "", priority = false, sizes = "(max-width: 640px) 50vw, 320px" }: { artName: ArtName; alt: string; className?: string; priority?: boolean; sizes?: string }) {
  const item = art[artName];

  return (
    <Image
      src={item.src}
      alt={alt}
      width={item.width}
      height={item.height}
      priority={priority}
      loading={priority ? "eager" : "lazy"}
      sizes={sizes}
      className={className}
    />
  );
}
