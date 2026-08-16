"use client";

import { useEffect, useRef, useState } from "react";

const themeMusicSource = "/triple-agent/audio/triple_agent_music__93.wav";
const musicPreferenceKey = "triple-agent:music-enabled";
const themeMusicVolume = 0.32;

function readMusicPreference() {
  try {
    return window.localStorage.getItem(musicPreferenceKey) !== "off";
  } catch {
    return true;
  }
}

function saveMusicPreference(enabled: boolean) {
  try {
    window.localStorage.setItem(musicPreferenceKey, enabled ? "on" : "off");
  } catch {
    // Storage can be unavailable in private browsing contexts.
  }
}

export function ThemeMusic() {
  const audioRef = useRef<HTMLAudioElement>(null);
  const enabledRef = useRef(true);
  const userGestureRef = useRef(false);
  const resumeTimerRef = useRef<number | undefined>(undefined);
  const [musicEnabled, setMusicEnabled] = useState(true);
  const [isPlaying, setIsPlaying] = useState(false);

  function startMusic() {
    const audio = audioRef.current;
    if (!audio || !enabledRef.current) return;

    userGestureRef.current = true;
    audio.volume = themeMusicVolume;
    void audio.play().catch(() => {
      // Browsers can still reject playback when no usable gesture was received.
      setIsPlaying(false);
    });
  }

  function resumeIfUnexpectedlyPaused() {
    const audio = audioRef.current;
    if (!audio || !enabledRef.current || !userGestureRef.current || !audio.paused) return;
    if (resumeTimerRef.current !== undefined) window.clearTimeout(resumeTimerRef.current);
    resumeTimerRef.current = window.setTimeout(() => {
      resumeTimerRef.current = undefined;
      if (enabledRef.current && audio.paused) {
        void audio.play().catch(() => setIsPlaying(false));
      }
    }, 100);
  }

  useEffect(() => {
    const enabled = readMusicPreference();
    enabledRef.current = enabled;
    // Keep the server/client initial render identical, then synchronize the
    // browser preference after the effect has mounted.
    const syncTimer = window.setTimeout(() => setMusicEnabled(enabled), 0);
    return () => window.clearTimeout(syncTimer);
  }, []);

  useEffect(() => {
    const handleGesture = () => startMusic();
    const handleVisibilityChange = () => {
      if (document.visibilityState === "visible" && userGestureRef.current) startMusic();
    };

    document.addEventListener("pointerdown", handleGesture, true);
    document.addEventListener("keydown", handleGesture, true);
    document.addEventListener("visibilitychange", handleVisibilityChange);

    return () => {
      document.removeEventListener("pointerdown", handleGesture, true);
      document.removeEventListener("keydown", handleGesture, true);
      document.removeEventListener("visibilitychange", handleVisibilityChange);
      if (resumeTimerRef.current !== undefined) window.clearTimeout(resumeTimerRef.current);
    };
  }, []);

  function toggleMusic() {
    const nextEnabled = !enabledRef.current;
    enabledRef.current = nextEnabled;
    setMusicEnabled(nextEnabled);
    saveMusicPreference(nextEnabled);

    if (nextEnabled) {
      startMusic();
    } else {
      audioRef.current?.pause();
      setIsPlaying(false);
    }
  }

  return (
    <>
      <audio
        ref={audioRef}
        aria-hidden="true"
        loop
        onPause={() => {
          setIsPlaying(false);
          resumeIfUnexpectedlyPaused();
        }}
        onPlay={() => setIsPlaying(true)}
        playsInline
        preload="none"
        src={themeMusicSource}
      />
      <button
        className="ta-tab min-w-10 px-1"
        aria-label={musicEnabled ? "Turn off theme music" : "Turn on theme music"}
        aria-pressed={musicEnabled}
        data-active={musicEnabled && isPlaying}
        onClick={toggleMusic}
        title={musicEnabled ? "Theme music on" : "Theme music off"}
        type="button"
      >
        <span aria-hidden="true" className="text-xl leading-none">{musicEnabled ? "♫" : "♩"}</span>
      </button>
    </>
  );
}
