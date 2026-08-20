import { InstallApp } from "@/features/pwa/install-app";
import { TripleAgentGame } from "@/features/triple-agent";

export default function Home() {
  return (
    <>
      <TripleAgentGame />
      <InstallApp />
    </>
  );
}
