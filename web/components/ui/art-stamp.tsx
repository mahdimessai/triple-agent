import Image from "next/image";
import { art, type ArtName } from "@/components/triple-agent/asset-registry";

export function ArtStamp({ artName, alt, className = "", priority = false }: { artName: ArtName; alt: string; className?: string; priority?: boolean }) {
  const item = art[artName];

  return (
    <Image
      src={item.src}
      alt={alt}
      width={item.width}
      height={item.height}
      priority={priority}
      className={className}
    />
  );
}
