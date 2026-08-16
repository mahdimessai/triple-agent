import { ArtStamp } from "@/components/ui/art-stamp";
import { ThemeMusic } from "@/components/triple-agent/theme-music";
import type { ScreenId } from "@/features/triple-agent/model/screen";

export function GameHeader({ screen, toggleSettings, liveSession }: { screen: ScreenId; toggleSettings: () => void; liveSession: boolean }) {
  return <header className="ta-header border-b-4 border-black bg-ta-orange-deep px-3 py-3 text-ta-paper lg:px-5"><div className="ta-header-inner"><div className="flex min-w-0 items-center gap-3"><h1 className="ta-display text-2xl">Triple Agent</h1></div><nav className="flex items-center gap-2" aria-label="Live room controls"><ThemeMusic />{liveSession ? <button className="ta-tab" aria-label={screen === "settings" ? "Close room settings" : "Room settings"} aria-expanded={screen === "settings"} data-active={screen === "settings"} onClick={toggleSettings} type="button"><ArtStamp artName="settings" alt="" className="h-5 w-8 object-contain" /></button> : null}</nav></div></header>;
}
