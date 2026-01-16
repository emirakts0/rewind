import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/** Format seconds as MM:SS (e.g. 90 → "1:30") */
export function formatTime(seconds: number): string {
  const mins = Math.floor(seconds / 60)
  const secs = seconds % 60
  return `${mins}:${secs.toString().padStart(2, '0')}`
}

/** Format seconds for buffer display (e.g. 90 → "1:30", 60 → "1") */
export function formatBufferDisplay(seconds: number): string {
  if (seconds >= 60) {
    const mins = Math.floor(seconds / 60)
    const secs = seconds % 60
    return secs > 0 ? `${mins}:${secs.toString().padStart(2, '0')}` : `${mins}`
  }
  return seconds.toString()
}

/** Get unit label for buffer display */
export function getBufferUnit(seconds: number): string {
  return seconds >= 60 ? 'min' : 'sec'
}

/** Format bytes as human readable (e.g. 1536 → "1.5 KB") */
export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
}
