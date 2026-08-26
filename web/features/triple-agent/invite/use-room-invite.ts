"use client";

import { useEffect, useRef, useState, type Dispatch, type SetStateAction } from "react";
import type { UseRoomResult } from "../session/room-state";

export type RoomInvite = { code: string; host?: string };

function legacyCopy(value: string): boolean {
  try {
    const input = document.createElement("textarea");
    input.value = value;
    input.setAttribute("readonly", "true");
    input.style.position = "fixed";
    input.style.opacity = "0";
    document.body.appendChild(input);
    input.select();
    const copied = document.execCommand("copy");
    input.remove();
    return copied;
  } catch {
    return false;
  }
}

async function copyText(value: string): Promise<boolean> {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value);
      return true;
    } catch {
      // Clipboard permission and focus failures can still succeed through the fallback.
    }
  }
  return legacyCopy(value);
}

function inviteUrl(room: UseRoomResult, code: string): string {
  const me = room.projection?.public.players.find((player) => player.id === room.projection?.private.player_id);
  const host = me?.name ? `&host=${encodeURIComponent(me.name)}` : "";
  return `${window.location.origin}${window.location.pathname}?join=${code}${host}`;
}

export function useRoomInvite({
  room,
  setJoinCode,
}: {
  room: UseRoomResult;
  setJoinCode: Dispatch<SetStateAction<string>>;
}) {
  const [invite, setInvite] = useState<RoomInvite | null>(null);
  const [copiedRoomCode, setCopiedRoomCode] = useState(false);
  const [linkShared, setLinkShared] = useState(false);
  const [copyError, setCopyError] = useState<string | null>(null);
  const copyTimerRef = useRef<number | null>(null);
  const shareTimerRef = useRef<number | null>(null);

  useEffect(() => () => {
    if (copyTimerRef.current !== null) window.clearTimeout(copyTimerRef.current);
    if (shareTimerRef.current !== null) window.clearTimeout(shareTimerRef.current);
  }, []);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const code = (params.get("join") ?? "").toUpperCase().replace(/[^A-Z0-9]/g, "").slice(0, 6);
    if (code.length !== 6) return;
    const host = (params.get("host") ?? "").replace(/[^\p{L}\p{N} '._-]/gu, "").trim().slice(0, 24);
    const syncTimer = window.setTimeout(() => {
      setJoinCode(code);
      setInvite({ code, host: host || undefined });
    }, 0);
    return () => window.clearTimeout(syncTimer);
  }, [setJoinCode]);

  function resetFeedback(): void {
    setCopyError(null);
    setCopiedRoomCode(false);
    setLinkShared(false);
  }

  function dismissInvite(): void {
    setInvite(null);
    setJoinCode("");
    window.history.replaceState(null, "", window.location.pathname);
  }

  async function copyRoomCode(): Promise<void> {
    const code = room.identity?.join_code;
    if (!code) return;
    if (!(await copyText(code))) {
      setCopiedRoomCode(false);
      setCopyError("Could not copy the room code");
      return;
    }
    setCopyError(null);
    setCopiedRoomCode(true);
    if (copyTimerRef.current !== null) window.clearTimeout(copyTimerRef.current);
    copyTimerRef.current = window.setTimeout(() => {
      copyTimerRef.current = null;
      setCopiedRoomCode(false);
    }, 1600);
  }

  async function shareRoomLink(): Promise<void> {
    const code = room.identity?.join_code;
    if (!code) return;
    const url = inviteUrl(room, code);
    if (navigator.share) {
      try {
        await navigator.share({ title: "Triple Agent", text: `Join my Triple Agent room: ${code}`, url });
        return;
      } catch {
        // A cancelled or unavailable share sheet falls through to copying.
      }
    }
    if (!(await copyText(url))) {
      setCopyError("Could not copy the invite link");
      return;
    }
    setCopyError(null);
    setLinkShared(true);
    if (shareTimerRef.current !== null) window.clearTimeout(shareTimerRef.current);
    shareTimerRef.current = window.setTimeout(() => {
      shareTimerRef.current = null;
      setLinkShared(false);
    }, 1600);
  }

  return {
    invite,
    copiedRoomCode,
    linkShared,
    copyError,
    dismissInvite,
    resetFeedback,
    copyRoomCode,
    shareRoomLink,
  };
}
