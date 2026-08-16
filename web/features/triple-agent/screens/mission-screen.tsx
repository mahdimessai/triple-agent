import { ArtStamp } from "@/components/ui/art-stamp";
import { InkButton } from "@/components/ui/ink-button";
import { PaperTitle } from "@/components/ui/paper-title";

export function MissionScreen({ onNext }: { onNext: () => void }) {
  return <div className="ta-rise ta-screen"><PaperTitle>Mission briefing</PaperTitle><div className="ta-paper relative overflow-hidden p-5"><div className="absolute -right-10 -top-10 opacity-15"><ArtStamp artName="passport" alt="" className="h-48 w-auto" /></div><p className="ta-condensed relative max-w-[32rem] text-lg leading-tight">Five players are in play. Two work for VIRUS. The Service must identify and imprison a VIRUS player before the network disappears.</p><div className="relative mt-5 border-t-2 border-black/25 pt-4"><p className="ta-condensed text-sm leading-tight">The server will reveal the next state to every connected player. Private facts appear only on the client they belong to.</p></div></div><div className="flex items-center justify-between gap-3"><span className="ta-condensed text-xs tracking-[0.16em]">ALL PLAYERS READY</span><InkButton onClick={onNext}>Continue</InkButton></div></div>;
}
