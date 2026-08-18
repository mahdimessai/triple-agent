"use client";

import { memo, useMemo, useState } from "react";
import type { RoomProjection } from "@/components/triple-agent/server-client";
import { InkButton } from "@/components/ui/ink-button";
import type { CommandSender } from "@/features/triple-agent/model/screen";
import { SettingsPanel } from "./settings-screen";
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

// Memoized Roster Column so it never re-renders when timer or settings change
const LobbyRosterPanel = memo(function LobbyRosterPanel({
    roomCode,
    roomCodeCopied,
    copyRoomCode,
    livePlayers,
    hostId,
    selfId,
    minPlayers,
    liveSession,
    isHost,
    canReady,
    isReady,
    readyLoading,
    startLoading,
    canStart,
    leaving,
    error,
    blockedReason,
    roster,
    displayNames,
    readyCount,
    missingPlayers,
    onReady,
    onStart,
    onTransferHost,
    onKickPlayer,
    onLeave,
}: {
    roomCode?: string;
    roomCodeCopied?: boolean;
    copyRoomCode?: () => void;
    livePlayers?: RoomProjection["public"]["players"];
    hostId?: string;
    selfId?: string;
    minPlayers: number;
    liveSession: boolean;
    isHost?: boolean;
    canReady?: boolean;
    isReady?: boolean;
    readyLoading: boolean;
    startLoading: boolean;
    canStart: boolean;
    leaving: boolean;
    error?: string;
    blockedReason?: string;
    roster: Array<{ id: string; name: string; seat: number; ready: boolean; connected: boolean; vote_submitted: boolean }>;
    displayNames: string[];
    readyCount: number;
    missingPlayers: number;
    onReady?: () => void;
    onStart: () => void;
    onTransferHost?: (targetId: string) => void;
    onKickPlayer?: (targetId: string) => void;
    onLeave?: () => void;
}) {
    return (
        <div className="w-full space-y-3">
            {roomCode ? (
                <div className="ta-paper flex flex-wrap items-center justify-between gap-x-4 gap-y-3 px-4 py-3">
                    <div className="min-w-0">
                        <p className="ta-condensed text-[0.65rem] tracking-[0.2em] text-black/60">ROOM CODE</p>
                        <p className="ta-display truncate text-[clamp(1.75rem,7vw,2.75rem)] leading-none tracking-[0.12em]">{roomCode}</p>
                    </div>
                    <button className="ta-secondary-button shrink-0 px-5" onClick={copyRoomCode} type="button" aria-label={`Copy room code ${roomCode}`}>
                        {roomCodeCopied ? "COPIED" : "COPY CODE"}
                    </button>
                    <p className="ta-condensed w-full text-xs leading-tight text-black/60">Share this code so the rest of the table can join.</p>
                </div>
            ) : null}

            <div className="grid gap-2">
                {liveSession && !livePlayers ? <p className="ta-condensed px-1 py-3 text-sm text-white">Synchronizing the room with the server…</p> : null}
                {roster.map((player, index) => {
                    const isPlayerHost = player.id === (hostId ?? livePlayers?.find((candidate) => candidate.seat === 1)?.id);
                    const canManage = isHost && player.id !== selfId;
                    return (
                        <div className="ta-paper flex flex-col gap-1.5 p-2.5" key={player.id}>
                            <div className="flex items-center justify-between gap-2">
                                <div className="flex min-w-0 items-center gap-1.5">
                                    <span className="ta-condensed truncate text-base font-bold text-ta-ink">{player.seat}. {displayNames[index]}</span>
                                    {player.id === selfId ? <span className="ta-ready-badge shrink-0" data-connected="true" data-ready="true">YOU</span> : null}
                                    {isPlayerHost ? <span className="ta-condensed shrink-0 text-[0.6rem] font-bold tracking-wider text-black/50">HOST</span> : null}
                                </div>
                                <span className="ta-ready-badge shrink-0" data-connected={player.connected} data-ready={player.ready}>
                                    {player.connected ? (player.ready ? "READY" : "NOT READY") : "OFFLINE"}
                                </span>
                            </div>
                            {canManage ? (
                                <div className="flex items-center justify-end gap-1.5 pt-1 border-t border-black/10">
                                    <button
                                        className="ta-secondary-button !min-h-0 border-2 border-black px-2 py-0.5 text-[0.68rem] font-bold tracking-wider uppercase transition-all hover:bg-ta-teal active:translate-x-0.5 active:translate-y-0.5"
                                        onClick={() => onTransferHost?.(player.id)}
                                        type="button"
                                        title="Make host"
                                    >
                                        Make Host
                                    </button>
                                    <button
                                        className="ta-secondary-button !min-h-0 border-2 border-black !bg-ta-orange-deep !text-ta-paper px-2 py-0.5 text-[0.68rem] font-bold tracking-wider uppercase transition-all hover:!bg-ta-red active:translate-x-0.5 active:translate-y-0.5"
                                        onClick={() => onKickPlayer?.(player.id)}
                                        type="button"
                                        title="Kick player"
                                    >
                                        Kick
                                    </button>
                                </div>
                            ) : null}
                        </div>
                    );
                })}
                {liveSession && missingPlayers > 0 ? Array.from({ length: missingPlayers }, (_, index) => (
                    <div className="ta-empty-seat" key={`empty-${index}`}>
                        <span className="ta-condensed text-xs tracking-[0.14em]">{roster.length + index + 1}. WAITING FOR A PLAYER</span>
                    </div>
                )) : null}
            </div>

            {liveSession ? (
                <div className="ta-paper flex items-center justify-between gap-3 px-4 py-2.5">
                    <span className="ta-condensed text-xs tracking-[0.16em]">READY STATUS</span>
                    <span className="ta-condensed text-sm font-bold">{readyCount} / {roster.length} READY · {minPlayers} NEEDED</span>
                </div>
            ) : null}

            {error ? <p className="ta-condensed text-sm text-white">{error}</p> : null}

            {liveSession && canReady ? (
                <InkButton className="w-full" onClick={onReady} disabled={startLoading} loading={readyLoading} loadingLabel="Saving…">
                    {isReady ? "Not ready" : "I'm ready"}
                </InkButton>
            ) : null}

            {liveSession && !isHost ? (
                <p className="ta-condensed text-center text-xs text-white/80">{blockedReason ?? "Waiting for the host to start the match"}</p>
            ) : (
                <div className="space-y-1.5">
                    <InkButton variant="orange" className="w-full" onClick={onStart} disabled={!canStart || readyLoading} loading={startLoading} loadingLabel="Starting match…">
                        Start match
                    </InkButton>
                    {liveSession && blockedReason ? <p className="ta-condensed text-center text-xs text-white/80">{blockedReason}</p> : null}
                </div>
            )}

            {liveSession ? (
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
});

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
    projection,
    timerEnabled = true,
    setTimerEnabled,
    onCommand,
    pendingCommand,
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
    projection?: RoomProjection;
    timerEnabled?: boolean;
    setTimerEnabled?: (value: boolean) => void;
    onCommand?: CommandSender;
    pendingCommand?: string;
}) {
    const [mobileTab, setMobileTab] = useState<"roster" | "settings">("roster");

    const roster = useMemo(() => {
        if (livePlayers?.length) return livePlayers;
        if (liveSession) return [];
        return players.map((player, index) => ({
            id: player.name,
            name: player.name,
            seat: index + 1,
            ready: player.state === "READY",
            connected: true,
            vote_submitted: false,
        }));
    }, [livePlayers, liveSession]);

    const displayNames = useMemo(() => disambiguate(roster), [roster]);
    const readyCount = useMemo(() => roster.filter((player) => player.ready).length, [roster]);
    const missingPlayers = Math.max(0, minPlayers - roster.length);
    const everyoneReady = roster.length > 0 && roster.every((player) => player.ready);
    const canStart = !liveSession || Boolean(isHost && missingPlayers === 0 && everyoneReady);
    const enabledOpsCount = projection?.public.settings.enabled_operations?.length ?? 12;

    const blockedReason = missingPlayers > 0
        ? `${missingPlayers} more ${missingPlayers === 1 ? "player" : "players"} needed to start`
        : !everyoneReady
        ? `Waiting for ${roster.length - readyCount} ${roster.length - readyCount === 1 ? "player" : "players"} to ready up`
        : undefined;

    return (
        <div className="ta-rise ta-screen ta-screen--wide ta-screen--lobby">
            {/* Mobile View Segmented Switch */}
            <div className="flex lg:hidden w-full border-2 border-black bg-ta-paper p-1 gap-1.5 shadow-[3px_3px_0_var(--ta-shadow)]">
                <button
                    type="button"
                    className={`flex-1 py-2 text-xs font-bold tracking-widest uppercase transition-all ${
                        mobileTab === "roster"
                            ? "bg-ta-ink text-ta-paper border-2 border-black shadow-[2px_2px_0_var(--ta-shadow)]"
                            : "bg-transparent text-black/75 hover:bg-black/5"
                    }`}
                    onClick={() => setMobileTab("roster")}
                >
                    ROSTER ({readyCount}/{roster.length})
                </button>
                <button
                    type="button"
                    className={`flex-1 py-2 text-xs font-bold tracking-widest uppercase transition-all ${
                        mobileTab === "settings"
                            ? "bg-ta-ink text-ta-paper border-2 border-black shadow-[2px_2px_0_var(--ta-shadow)]"
                            : "bg-transparent text-black/75 hover:bg-black/5"
                    }`}
                    onClick={() => setMobileTab("settings")}
                >
                    SETTINGS ({enabledOpsCount} OPS)
                </button>
            </div>

            <div className="grid w-full gap-5 lg:grid-cols-12 lg:items-start">
                {/* Left Column: Room Info, Player Roster & Actions */}
                <div className={`w-full lg:col-span-4 lg:sticky lg:top-4 ${mobileTab !== "roster" ? "hidden lg:block" : "block"}`}>
                    <LobbyRosterPanel
                        roomCode={roomCode}
                        roomCodeCopied={roomCodeCopied}
                        copyRoomCode={copyRoomCode}
                        livePlayers={livePlayers}
                        hostId={hostId}
                        selfId={selfId}
                        minPlayers={minPlayers}
                        liveSession={liveSession}
                        isHost={isHost}
                        canReady={canReady}
                        isReady={isReady}
                        readyLoading={readyLoading}
                        startLoading={startLoading}
                        canStart={canStart}
                        leaving={leaving}
                        error={error}
                        blockedReason={blockedReason}
                        roster={roster}
                        displayNames={displayNames}
                        readyCount={readyCount}
                        missingPlayers={missingPlayers}
                        onReady={onReady}
                        onStart={onStart}
                        onTransferHost={onTransferHost}
                        onKickPlayer={onKickPlayer}
                        onLeave={onLeave}
                    />
                </div>

                {/* Right Column: Live Settings */}
                <div className={`w-full lg:col-span-8 ${mobileTab !== "settings" ? "hidden lg:block" : "block"}`}>
                    <SettingsPanel
                        timerEnabled={timerEnabled}
                        setTimerEnabled={setTimerEnabled ?? (() => {})}
                        projection={projection}
                        liveSession={liveSession}
                        isHost={isHost}
                        onCommand={onCommand}
                        pending={Boolean(pendingCommand)}
                        error={error}
                        showHeader={false}
                    />
                </div>
            </div>
        </div>
    );
}