import Image from "next/image";
import { art, type ArtName } from "./assets";

export { InkButton, type InkButtonProps } from "@/components/ui/ink-button";
export { PaperTitle } from "@/components/ui/paper-title";

export type ArtStampProps = {
  artName: ArtName;
  alt?: string;
  className?: string;
  priority?: boolean;
  sizes?: string;
};

// ArtStamp stays feature-owned because the art registry is specific to Triple Agent.
export function ArtStamp({ artName, alt = "", className = "", priority = false, sizes = "(max-width: 640px) 50vw, 320px" }: ArtStampProps) {
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
