import { Game } from "@/features/triple-agent/game";
import { InstallApp } from "./install-app";

export default function Home() {
  return (
    <>
      <Game />
      <InstallApp />
    </>
  );
}
