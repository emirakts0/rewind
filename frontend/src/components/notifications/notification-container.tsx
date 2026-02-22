
import { AnimatePresence, motion } from 'framer-motion'
import { X, CheckCircle2, AlertTriangle, AlertCircle, Info } from 'lucide-react'
import {
    useNotifications,
    type Notification,
    type NotificationType,
    type NotificationPosition,
} from './notification-store'
import { cn } from '@/lib/utils'

// ─── Notification Icon ───────────────────────────────────────────────────────

const iconMap: Record<NotificationType, React.ReactNode> = {
    success: <CheckCircle2 className="w-4 h-4" />,
    error: <AlertCircle className="w-4 h-4" />,
    warning: <AlertTriangle className="w-4 h-4" />,
    info: <Info className="w-4 h-4" />,
}

const colorMap: Record<NotificationType, { bg: string; border: string; icon: string; text: string }> = {
    success: {
        bg: 'bg-[hsl(152_80%_8%)]',
        border: 'border-[hsl(152_60%_20%)]',
        icon: 'text-[hsl(152_70%_55%)]',
        text: 'text-[hsl(152_60%_75%)]',
    },
    error: {
        bg: 'bg-[hsl(0_70%_10%)]',
        border: 'border-[hsl(0_60%_22%)]',
        icon: 'text-[hsl(0_80%_65%)]',
        text: 'text-[hsl(0_60%_75%)]',
    },
    warning: {
        bg: 'bg-[hsl(39_70%_9%)]',
        border: 'border-[hsl(39_60%_22%)]',
        icon: 'text-[hsl(39_87%_63%)]',
        text: 'text-[hsl(39_60%_75%)]',
    },
    info: {
        bg: 'bg-[hsl(220_50%_10%)]',
        border: 'border-[hsl(220_40%_22%)]',
        icon: 'text-[hsl(220_60%_65%)]',
        text: 'text-[hsl(220_40%_75%)]',
    },
}

// ─── Animation Variants ──────────────────────────────────────────────────────

function getSlideDirection(position: NotificationPosition) {
    return position.includes('left') ? -1 : 1
}



// ─── Single Notification Item ────────────────────────────────────────────────

function NotificationItem({
    notification,
    position,
    durationMs,
    onDismiss,
}: {
    notification: Notification
    position: NotificationPosition
    durationMs: number
    onDismiss: (id: string) => void
}) {
    const colors = colorMap[notification.type]
    const slideDir = getSlideDirection(position)

    return (
        <motion.div
            layout
            initial={{ x: slideDir * 340, opacity: 0, scale: 0.85 }}
            animate={{ x: 0, opacity: 1, scale: 1 }}
            exit={{ x: slideDir * 340, opacity: 0, scale: 0.9 }}
            transition={{
                type: 'spring',
                stiffness: 380,
                damping: 30,
                mass: 0.8,
            }}
            className={cn(
                'group relative w-[300px] rounded-xl border shadow-2xl backdrop-blur-md overflow-hidden cursor-pointer',
                'transition-colors duration-200',
                colors.bg,
                colors.border,
            )}
            onClick={() => onDismiss(notification.id)}
            role="alert"
            aria-live="polite"
        >
            {/* Progress bar */}
            <div className="absolute top-0 left-0 right-0 h-[2px] overflow-hidden rounded-t-xl">
                <motion.div
                    initial={{ scaleX: 1 }}
                    animate={{ scaleX: 0 }}
                    transition={{ duration: durationMs / 1000, ease: 'linear' }}
                    className={cn(
                        'h-full origin-left',
                        notification.type === 'success' && 'bg-[hsl(152_70%_45%)]',
                        notification.type === 'error' && 'bg-[hsl(0_80%_55%)]',
                        notification.type === 'warning' && 'bg-[hsl(39_87%_55%)]',
                        notification.type === 'info' && 'bg-[hsl(220_60%_55%)]',
                    )}
                />
            </div>

            <div className="flex items-start gap-3 px-4 py-3">
                {/* Icon */}
                <div className={cn('mt-0.5 flex-shrink-0', colors.icon)}>
                    {iconMap[notification.type]}
                </div>

                {/* Content */}
                <div className="flex-1 min-w-0">
                    <p className={cn('text-[13px] font-semibold leading-tight', colors.text)}>
                        {notification.title}
                    </p>
                    {notification.description && (
                        <p className={cn('text-[11px] mt-1 leading-snug opacity-70', colors.text)}>
                            {notification.description}
                        </p>
                    )}
                </div>

                {/* Close button */}
                <button
                    onClick={(e) => {
                        e.stopPropagation()
                        onDismiss(notification.id)
                    }}
                    className={cn(
                        'flex-shrink-0 mt-0.5 p-0.5 rounded-md opacity-0 group-hover:opacity-70 hover:!opacity-100 transition-opacity',
                        colors.text,
                    )}
                    aria-label="Close notification"
                >
                    <X className="w-3.5 h-3.5" />
                </button>
            </div>
        </motion.div>
    )
}

// ─── Notification Container ──────────────────────────────────────────────────

export function NotificationContainer() {
    const { notifications, config, dismiss } = useNotifications()

    const positionClasses: Record<NotificationPosition, string> = {
        'top-left': 'top-3 left-3',
        'top-right': 'top-3 right-3',
        'bottom-left': 'bottom-3 left-3',
        'bottom-right': 'bottom-3 right-3',
    }

    const isBottom = config.position.includes('bottom')

    return (
        <div
            className={cn(
                'fixed z-[9999] flex flex-col gap-2 pointer-events-none',
                positionClasses[config.position],
            )}
            style={{
                flexDirection: isBottom ? 'column-reverse' : 'column',
            }}
        >
            <AnimatePresence mode="popLayout">
                {notifications.map(notif => (
                    <div key={notif.id} className="pointer-events-auto">
                        <NotificationItem
                            notification={notif}
                            position={config.position}
                            durationMs={config.durationMs}
                            onDismiss={dismiss}
                        />
                    </div>
                ))}
            </AnimatePresence>
        </div>
    )
}
