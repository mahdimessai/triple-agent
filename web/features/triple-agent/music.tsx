"use client";

import { useEffect, useRef, useState } from "react";

const THEME_MUSIC_SOURCE = "/triple-agent/audio/triple_agent_music__93.wav";
const MUSIC_PREFERENCE_KEY = "triple-agent:music-enabled";
const THEME_MUSIC_VOLUME = 0.32;

function readMusicPreference(): boolean {
  try {
    return window.localStorage.getItem(MUSIC_PREFERENCE_KEY) !== "off";
  } catch {
    return true;
  }
}

function saveMusicPreference(enabled: boolean): void {
  try {
    window.localStorage.setItem(MUSIC_PREFERENCE_KEY, enabled ? "on" : "off");
  } catch {
    // Preference persistence is optional.
  }
}

export function ThemeMusic() {
  const audioRef = useRef<HTMLAudioElement>(null);
  const enabledRef = useRef(true);
  const userGestureRef = useRef(false);
  const resumeTimerRef = useRef<number | null>(null);
  const [enabled, setEnabled] = useState(true);
  const [playing, setPlaying] = useState(false);

  function start(): void {
    const audio = audioRef.current;
    if (!audio || !enabledRef.current) return;
    userGestureRef.current = true;
    audio.volume = THEME_MUSIC_VOLUME;
    void audio.play().catch(() => setPlaying(false));
  }

  function resumeUnexpectedPause(): void {
    const audio = audioRef.current;
    if (!audio || !enabledRef.current || !userGestureRef.current || !audio.paused) return;
    if (resumeTimerRef.current !== null) window.clearTimeout(resumeTimerRef.current);
    resumeTimerRef.current = window.setTimeout(() => {
      resumeTimerRef.current = null;
      if (enabledRef.current && audio.paused) void audio.play().catch(() => setPlaying(false));
    }, 100);
  }

  useEffect(() => {
    const preference = readMusicPreference();
    enabledRef.current = preference;
    const timer = window.setTimeout(() => setEnabled(preference), 0);
    return () => window.clearTimeout(timer);
  }, []);

  useEffect(() => {
    const onGesture = () => start();
    const onVisibility = () => {
      if (document.visibilityState === "visible" && userGestureRef.current) start();
    };

    document.addEventListener("pointerdown", onGesture, true);
    document.addEventListener("keydown", onGesture, true);
    document.addEventListener("visibilitychange", onVisibility);
    return () => {
      document.removeEventListener("pointerdown", onGesture, true);
      document.removeEventListener("keydown", onGesture, true);
      document.removeEventListener("visibilitychange", onVisibility);
      if (resumeTimerRef.current !== null) window.clearTimeout(resumeTimerRef.current);
    };
  }, []);

  function toggle(): void {
    const next = !enabledRef.current;
    enabledRef.current = next;
    setEnabled(next);
    saveMusicPreference(next);
    if (next) start();
    else {
      audioRef.current?.pause();
      setPlaying(false);
    }
  }

  return (
    <>
      <audio
        ref={audioRef}
        aria-hidden="true"
        loop
        onPause={() => {
          setPlaying(false);
          resumeUnexpectedPause();
        }}
        onPlay={() => setPlaying(true)}
        playsInline
        preload="none"
        src={THEME_MUSIC_SOURCE}
      />
      <button
        className="ta-tab min-w-10 px-1"
        aria-label={enabled ? "Turn off theme music" : "Turn on theme music"}
        aria-pressed={enabled}
        data-active={enabled && playing}
        onClick={toggle}
        title={enabled ? "Theme music on" : "Theme music off"}
        type="button"
      >
        <span aria-hidden="true" className="text-xl leading-none">{enabled ? "♫" : "♩"}</span>
      </button>
    </>
  );
}
