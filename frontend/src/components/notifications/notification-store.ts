import { useCallback, useSyncExternalStore } from 'react'
import { api } from '@/lib/wails'

// ─── Types ───────────────────────────────────────────────────────────────────

export type NotificationType = 'success' | 'error' | 'warning' | 'info'

export type NotificationPosition = 'top-left' | 'top-right' | 'bottom-left' | 'bottom-right'

export interface NotificationConfig {
    enabled: boolean
    onlyErrors: boolean       // show only error/warning types
    position: NotificationPosition
    durationMs: number        // auto-dismiss duration in ms
}

export interface Notification {
    id: string
    type: NotificationType
    title: string
    description?: string
    createdAt: number
    exiting?: boolean         // true when slide-out animation is playing
}

// ─── Constants ───────────────────────────────────────────────────────────────

const MAX_VISIBLE = 3
const EXIT_ANIMATION_MS = 400

const DEFAULT_CONFIG: NotificationConfig = {
    enabled: true,
    onlyErrors: false,
    position: 'top-left',
    durationMs: 3000,
}

// ─── Store ───────────────────────────────────────────────────────────────────

let notifications: Notification[] = []
let config: NotificationConfig = { ...DEFAULT_CONFIG }
let queue: Notification[] = []    // waiting list when MAX_VISIBLE is reached
let listeners: Set<() => void> = new Set()
let idCounter = 0

function emit() {
    currentSnapshot = {
        notifications,
        config,
        queueLength: queue.length,
    }

    if (isNotificationWindow) {
        if (notifications.length > 0) {
            api.showNotificationWindow(config.position || 'bottom-right')
        } else {
            api.hideNotificationWindow()
        }
    }

    listeners.forEach(fn => fn())
}

function promoteFromQueue() {
    while (queue.length > 0 && notifications.filter(n => !n.exiting).length < MAX_VISIBLE) {
        const next = queue.shift()!
        notifications = [...notifications, next]
        scheduleAutoDismiss(next.id)
    }
    emit()
}

function scheduleAutoDismiss(id: string) {
    const duration = config.durationMs
    setTimeout(() => {
        dismiss(id)
    }, duration)
}

// ─── Public API ──────────────────────────────────────────────────────────────

const isNotificationWindow = window.location.hash.includes('notification')

export function notify(type: NotificationType, title: string, description?: string) {
    if (!config.enabled) return
    if (config.onlyErrors && type !== 'error' && type !== 'warning') return

    if (!isNotificationWindow) {
        api.broadcastNotification(type, title, description)
        return
    }

    notifyLocal(type, title, description)
}

function notifyLocal(type: NotificationType, title: string, description?: string) {
    const id = `notif-${++idCounter}-${Date.now()}`
    const notification: Notification = {
        id,
        type,
        title,
        description,
        createdAt: Date.now(),
    }

    const visibleCount = notifications.filter(n => !n.exiting).length
    if (visibleCount >= MAX_VISIBLE) {
        queue = [...queue, notification]
        emit()
        return
    }

    notifications = [...notifications, notification]
    emit()
    scheduleAutoDismiss(id)
}

if (isNotificationWindow) {
    api.Events.On('notification:show', (e: any) => {
        const payload = e.data
        if (payload) {
            notifyLocal(payload.type, payload.title, payload.description)
        }
    })

    api.Events.On('notification:config-updated', (e: any) => {
        const payload = e.data
        if (payload) {
            config = { ...config, ...payload }
            emit()
        }
    })
}

export function dismiss(id: string) {
    const exists = notifications.find(n => n.id === id)
    if (!exists || exists.exiting) return

    // Mark as exiting for animation
    notifications = notifications.map(n =>
        n.id === id ? { ...n, exiting: true } : n
    )
    emit()

    // Remove after animation completes
    setTimeout(() => {
        notifications = notifications.filter(n => n.id !== id)
        promoteFromQueue()
    }, EXIT_ANIMATION_MS)
}

export function clearAll() {
    notifications = []
    queue = []
    emit()
}

export function getConfig(): NotificationConfig {
    return config
}

export function updateConfig(partial: Partial<NotificationConfig>) {
    config = { ...config, ...partial }
    emit()

    // Broadcast config changes to the notification window
    if (!isNotificationWindow) {
        api.broadcastConfigUpdate(partial)
    }
}

let currentSnapshot = {
    notifications,
    config,
    queueLength: queue.length,
}

export function getSnapshot() {
    return currentSnapshot
}

export function subscribe(listener: () => void): () => void {
    listeners.add(listener)
    return () => listeners.delete(listener)
}

// ─── React Hook ──────────────────────────────────────────────────────────────

export function useNotifications() {
    const snapshot = useSyncExternalStore(subscribe, getSnapshot, getSnapshot)

    const send = useCallback((type: NotificationType, title: string, description?: string) => {
        notify(type, title, description)
    }, [])

    const remove = useCallback((id: string) => {
        dismiss(id)
    }, [])

    const setConfig = useCallback((partial: Partial<NotificationConfig>) => {
        updateConfig(partial)
    }, [])

    return {
        notifications: snapshot.notifications,
        config: snapshot.config,
        queueLength: snapshot.queueLength,
        notify: send,
        dismiss: remove,
        clearAll,
        updateConfig: setConfig,
    }
}
