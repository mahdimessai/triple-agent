"use client";

import type { RoomProjection } from "@/components/triple-agent/server-client";
import { InkButton } from "@/components/ui/ink-button";
import { players } from "./fixtures";

const DEFAULT_MIN_PLAYERS = 5;

/**
 * Everyone defaults to the same name, so a table of "AGENT A"s is the norm
 * rather than the exception. Suffix the repeats so players can tell the roster
 * apart without anyone having to retype their name.
 */
function disambiguate(roster: Array<{ id: string; name: string }>) {
    const seen = new Map<string, number>();
    const totals = new Map<string, number>();
    for (const player of roster) totals.set(player.name, (totals.get(player.name) ?? 0) + 1);
    return roster.map((player) => {
        if ((totals.get(player.name) ?? 0) < 2) return player.name;
        const index = (seen.get(player.name) ?? 0) + 1;
        seen.set(player.name, index);
        return `${player.name} (${index})`;
    });
}

export function LobbyScreen({
                                roomCode,
                                roomCodeCopied,
                                copyRoomCode,
                                livePlayers,
                                hostId,
                                selfId,
                                minPlayers = DEFAULT_MIN_PLAYERS,
                                liveSession = false,
                                isHost,
                                canReady,
                                isReady,
                                readyLoading = false,
                                startLoading = false,
                                onReady,
                                onStart,
                                onTransferHost,
                                onKickPlayer,
                                onLeave,
                                leaving = false,
                                error,
                            }: {
    roomCode?: string;
    roomCodeCopied?: boolean;
    copyRoomCode?: () => void;
    livePlayers?: RoomProjection["public"]["players"];
    hostId?: string;
    selfId?: string;
    minPlayers?: number;
    liveSession?: boolean;
    isHost?: boolean;
    canReady?: boolean;
    isReady?: boolean;
    readyLoading?: boolean;
    startLoading?: boolean;
    onReady?: () => void;
    onStart: () => void;
    onTransferHost?: (targetId: string) => void;
    onKickPlayer?: (targetId: string) => void;
    onLeave?: () => void;
    leaving?: boolean;
    error?: string;
}) {
    const roster = livePlayers?.length ? livePlayers : liveSession ? [] : players.map((player, index) => ({ id: player.name, name: player.name, seat: index + 1, ready: player.state === "READY", connected: true, vote_submitted: false }));
    const isLive = liveSession;
    const displayNames = disambiguate(roster);
    const readyCount = roster.filter((player) => player.ready).length;
    const missingPlayers = Math.max(0, minPlayers - roster.length);
    const everyoneReady = roster.length > 0 && roster.every((player) => player.ready);
    const canStart = !isLive || Boolean(isHost && missingPlayers === 0 && everyoneReady);
    // A disabled button with no explanation reads as a broken game, so always say
    // what the room is still waiting for.
    const blockedReason = missingPlayers > 0
        ? `${missingPlayers} more ${missingPlayers === 1 ? "player" : "players"} needed to start`
        : !everyoneReady
            ? `Waiting for ${roster.length - readyCount} ${roster.length - readyCount === 1 ? "player" : "players"} to ready up`
            : undefined;

    return (
        <div className="ta-rise ta-screen ta-screen--lobby">
            {roomCode ? (
                <div className="ta-paper flex flex-wrap items-center justify-between gap-x-4 gap-y-3 px-4 py-3">
                    <div className="min-w-0">
                        <p className="ta-condensed text-[0.65rem] tracking-[0.2em] text-black/60">ROOM CODE</p>
                        <p className="ta-display truncate text-[clamp(1.75rem,7vw,2.75rem)] leading-none tracking-[0.12em]">{roomCode}</p>
                    </div>
                    <button className="ta-secondary-button shrink-0 px-5" onClick={copyRoomCode} type="button" aria-label={`Copy room code ${roomCode}`}>{roomCodeCopied ? "COPIED" : "COPY CODE"}</button>
                    <p className="ta-condensed w-full text-sm leading-tight text-black/60">Share this code so the rest of the table can join.</p>
                </div>
            ) : null}

            <div className="grid gap-2">
                {liveSession && !livePlayers ? <p className="ta-condensed px-1 py-3 text-sm text-white">Synchronizing the room with the server…</p> : null}
                {roster.map((player, index) => {
                    const isPlayerHost = player.id === (hostId ?? livePlayers?.find((candidate) => candidate.seat === 1)?.id);
                    const canManage = isHost && player.id !== selfId;
                    return (
                        <div className="ta-paper flex flex-col gap-2 p-3 sm:flex-row sm:items-center sm:justify-between" key={player.id}>
                            <div className="flex min-w-0 items-center gap-2 sm:gap-3">
                                <span className="ta-condensed truncate text-lg">{player.seat}. {displayNames[index]}</span>
                                {player.id === selfId ? <span className="ta-ready-badge" data-connected="true" data-ready="true">YOU</span> : null}
                                {isPlayerHost ? <span className="ta-condensed text-[0.6rem] tracking-[0.14em] text-black/50">HOST</span> : null}
                            </div>
                            <div className="flex flex-wrap items-center gap-2">
                                <span className="ta-ready-badge" data-connected={player.connected} data-ready={player.ready}>{player.connected ? (player.ready ? "READY" : "NOT READY") : "OFFLINE"}</span>
                                {canManage ? (
                                    <div className="flex items-center gap-1.5 ml-auto sm:ml-0">
                                        <button
                                            className="ta-secondary-button !min-h-0 border-2 border-black px-2.5 py-1 text-xs font-bold tracking-wider uppercase transition-all hover:bg-ta-teal active:translate-x-0.5 active:translate-y-0.5"
                                            onClick={() => onTransferHost?.(player.id)}
                                            type="button"
                                            title="Make host"
                                        >
                                            Make host
                                        </button>
                                        <button
                                            className="ta-secondary-button !min-h-0 border-2 border-black !bg-ta-orange-deep !text-ta-paper px-2.5 py-1 text-xs font-bold tracking-wider uppercase transition-all hover:!bg-ta-red active:translate-x-0.5 active:translate-y-0.5"
                                            onClick={() => onKickPlayer?.(player.id)}
                                            type="button"
                                            title="Kick player"
                                        >
                                            Kick
                                        </button>
                                    </div>
                                ) : null}
                            </div>
                        </div>
                    );
                })}
                {isLive && missingPlayers > 0 ? Array.from({ length: missingPlayers }, (_, index) => (
                    <div className="ta-empty-seat" key={`empty-${index}`}>
                        <span className="ta-condensed text-sm tracking-[0.14em]">{roster.length + index + 1}. WAITING FOR A PLAYER</span>
                    </div>
                )) : null}
            </div>
            {isLive ? <div className="ta-paper flex items-center justify-between gap-3 px-4 py-3"><span className="ta-condensed text-xs tracking-[0.16em]">READY STATUS</span><span className="ta-condensed text-sm">{readyCount} / {roster.length} READY · {minPlayers} NEEDED</span></div> : null}
            {error ? <p className="ta-condensed text-sm text-white">{error}</p> : null}
            {isLive && canReady ? <InkButton className="w-full" onClick={onReady} disabled={startLoading} loading={readyLoading} loadingLabel="Saving…">{isReady ? "Not ready" : "I'm ready"}</InkButton> : null}
            {isLive && !isHost ? <p className="ta-condensed text-center text-sm text-white/80">{blockedReason ?? "Waiting for the host to start the match"}</p> : (
                <>
                    <InkButton variant="orange" className="w-full" onClick={onStart} disabled={!canStart || readyLoading} loading={startLoading} loadingLabel="Starting match…">Start match</InkButton>
                    {isLive && blockedReason ? <p className="ta-condensed text-center text-sm text-white/80">{blockedReason}</p> : null}
                </>
            )}
            {isLive ? (
                <button
                    className="ta-secondary-button w-full"
                    onClick={onLeave}
                    disabled={leaving}
                    aria-busy={leaving ? true : undefined}
                    type="button"
                >
                    {leaving ? (
                        <span className="ta-button-loading">
                            <span className="ta-button-spinner" aria-hidden="true" />
                            Leaving…
                        </span>
                    ) : (
                        "Leave lobby"
                    )}
                </button>
            ) : null}
        </div>
    );
}