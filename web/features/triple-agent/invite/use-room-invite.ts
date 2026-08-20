"use client";

import { useEffect, useRef, useState } from "react";
import type { RoomIdentity, RoomProjection } from "../protocol";
import { buildRoomInviteUrl, parseRoomInvite, type RoomInvite } from "./invite";

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
      // Permission/focus failures still get the selection-based fallback.
    }
  }
  return legacyCopy(value);
}

export type RoomInviteController = {
  invite: RoomInvite | null;
  copiedRoomCode: boolean;
  linkShared: boolean;
  error: string | null;
  copyRoomCode(): Promise<void>;
  shareRoomLink(): Promise<void>;
  dismissInvite(): void;
  resetFeedback(): void;
};

export function useRoomInvite(identity: RoomIdentity | null, projection: RoomProjection | null): RoomInviteController {
  const [invite, setInvite] = useState<RoomInvite | null>(null);
  const [copiedRoomCode, setCopiedRoomCode] = useState(false);
  const [linkShared, setLinkShared] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const copyTimerRef = useRef<number | null>(null);
  const shareTimerRef = useRef<number | null>(null);

  useEffect(() => {
    setInvite(parseRoomInvite(window.location.search));
    return () => {
      if (copyTimerRef.current !== null) window.clearTimeout(copyTimerRef.current);
      if (shareTimerRef.current !== null) window.clearTimeout(shareTimerRef.current);
    };
  }, []);

  function resetFeedback(): void {
    setError(null);
    setCopiedRoomCode(false);
    setLinkShared(false);
  }

  function dismissInvite(): void {
    setInvite(null);
    resetFeedback();
    window.history.replaceState(null, "", window.location.pathname);
  }

  async function copyRoomCode(): Promise<void> {
    const code = identity?.join_code;
    if (!code) return;

    if (!(await copyText(code))) {
      setCopiedRoomCode(false);
      setError("Could not copy the room code");
      return;
    }

    setError(null);
    setCopiedRoomCode(true);
    if (copyTimerRef.current !== null) window.clearTimeout(copyTimerRef.current);
    copyTimerRef.current = window.setTimeout(() => {
      copyTimerRef.current = null;
      setCopiedRoomCode(false);
    }, 1600);
  }

  async function shareRoomLink(): Promise<void> {
    const code = identity?.join_code;
    if (!code) return;

    const me = projection?.public.players.find((player) => player.id === projection.private.player_id);
    const url = buildRoomInviteUrl(window.location.origin, window.location.pathname, code, me?.name);

    if (navigator.share) {
      try {
        await navigator.share({ title: "Triple Agent", text: `Join my Triple Agent room: ${code}`, url });
        return;
      } catch {
        // Cancellation or an unavailable share sheet falls through to copying.
      }
    }

    if (!(await copyText(url))) {
      setError("Could not copy the invite link");
      return;
    }

    setError(null);
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
    error,
    copyRoomCode,
    shareRoomLink,
    dismissInvite,
    resetFeedback,
  };
}
