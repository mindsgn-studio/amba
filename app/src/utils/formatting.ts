export function formatTime(totalSeconds: number): string {
  if (!Number.isFinite(totalSeconds) || totalSeconds < 0) {
    return '00:00';
  }
  const seconds = Math.floor(totalSeconds);
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  const mm = String(minutes).padStart(2, '0');
  const ss = String(remainingSeconds).padStart(2, '0');
  return `${mm}:${ss}`;
}

export function formatPercent(progress: number): string {
  const clamped = Math.max(0, Math.min(100, Math.round(progress)));
  return `${clamped}%`;
}
