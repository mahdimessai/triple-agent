import { ThemeMusic } from "@/components/triple-agent/theme-music";
import type { ScreenId } from "@/features/triple-agent/model/screen";

export function GameHeader({
                               screen,
                               goHome,
                           }: {
    screen: ScreenId;
    toggleSettings?: () => void;
    liveSession?: boolean;
    goHome?: () => void;
}) {
    const inGame = screen !== "title" && screen !== "join" && screen !== "lobby" && screen !== "settings";

    return (
        <header className="ta-header border-b-4 border-black bg-ta-orange-deep px-3 py-3 text-ta-paper lg:px-5">
            <div className="ta-header-inner">
                <div className="flex min-w-0 items-center gap-3">
                    {!inGame ? (
                        <button
                            type="button"
                            onClick={goHome}
                            className="ta-header-brand cursor-pointer"
                            aria-label="Return to main menu">
                            Triple Agent
                        </button>
                    ) : null}
                </div>

                <nav
                    className="flex items-center gap-2"
                    aria-label="Live room controls">
                    <ThemeMusic />
                </nav>
            </div>
        </header>
    );
}