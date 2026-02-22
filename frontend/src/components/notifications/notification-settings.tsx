import { Bell, BellOff, Clock, MapPin, AlertTriangle, Play } from 'lucide-react'
import { Switch } from '@/components/ui/switch'
import { Button } from '@/components/ui/button'
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'
import { useNotifications, type NotificationPosition } from './notification-store'
import { cn } from '@/lib/utils'
import type { Config } from '@/lib/wails'

const POSITION_OPTIONS: { value: NotificationPosition; label: string }[] = [
    { value: 'top-left', label: 'Top Left' },
    { value: 'top-right', label: 'Top Right' },
    { value: 'bottom-left', label: 'Bottom Left' },
    { value: 'bottom-right', label: 'Bottom Right' },
]

const DURATION_OPTIONS = [
    { value: 2000, label: '2 seconds' },
    { value: 3000, label: '3 seconds' },
    { value: 4000, label: '4 seconds' },
    { value: 5000, label: '5 seconds' },
    { value: 8000, label: '8 seconds' },
    { value: 10000, label: '10 seconds' },
]

export function NotificationSettings({ disabled, config, setConfig }: { disabled?: boolean, config: Config, setConfig: React.Dispatch<React.SetStateAction<Config>> }) {
    const { notify } = useNotifications()

    const handleUpdate = (updates: Partial<Config>) => {
        setConfig(prev => {
            const newConfig = { ...prev, ...updates };
            return newConfig;
        });
    }

    return (
        <div className={cn('space-y-3', disabled && 'opacity-50 pointer-events-none')}>
            {/* Enable / Disable */}
            <div className="flex items-center justify-between px-3 py-2 rounded-md border border-border/30 bg-secondary/5">
                <div className="flex items-center gap-2">
                    {config.notificationsEnabled
                        ? <Bell className="w-3.5 h-3.5 text-muted-foreground" />
                        : <BellOff className="w-3.5 h-3.5 text-muted-foreground" />
                    }
                    <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                        Notifications
                    </span>
                </div>
                <Switch
                    checked={config.notificationsEnabled}
                    onCheckedChange={(checked) => handleUpdate({ notificationsEnabled: checked })}
                    className="scale-90"
                />
            </div>

            {/* Only errors filter */}
            <div className={cn("flex items-center justify-between px-3 py-2 rounded-md border border-border/30 bg-secondary/5 transition-opacity", !config.notificationsEnabled && "opacity-50 pointer-events-none")}>
                <div className="flex items-center gap-2">
                    <AlertTriangle className="w-3.5 h-3.5 text-muted-foreground" />
                    <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                        Errors only
                    </span>
                </div>
                <Switch
                    checked={config.notificationsOnlyErrors}
                    onCheckedChange={(checked) => handleUpdate({ notificationsOnlyErrors: checked })}
                    disabled={!config.notificationsEnabled}
                    className="scale-90"
                />
            </div>

            <div className={cn("grid grid-cols-2 gap-4 transition-opacity", !config.notificationsEnabled && "opacity-50 pointer-events-none")}>
                {/* Position selector */}
                <div className="space-y-1.5">
                    <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider flex items-center gap-1.5">
                        <MapPin className="w-3 h-3" /> Position
                    </label>
                    <Select
                        value={config.notificationsPosition}
                        onValueChange={(v) => handleUpdate({ notificationsPosition: v as NotificationPosition })}
                        disabled={!config.notificationsEnabled}
                    >
                        <SelectTrigger className="h-9 bg-accent border-border/50">
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                            {POSITION_OPTIONS.map(opt => (
                                <SelectItem key={opt.value} value={opt.value}>
                                    {opt.label}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </div>

                {/* Duration selector */}
                <div className="space-y-1.5">
                    <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider flex items-center gap-1.5">
                        <Clock className="w-3 h-3" /> Display duration
                    </label>
                    <Select
                        value={config.notificationsDurationMs.toString()}
                        onValueChange={(v) => handleUpdate({ notificationsDurationMs: parseInt(v) })}
                        disabled={!config.notificationsEnabled}
                    >
                        <SelectTrigger className="h-9 bg-accent border-border/50">
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                            {DURATION_OPTIONS.map(opt => (
                                <SelectItem key={opt.value} value={opt.value.toString()}>
                                    {opt.label}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </div>
            </div>

            {/* Test button */}
            <Button
                variant="outline"
                size="sm"
                onClick={() => notify('info', 'Test notification', 'This is a preview of how notifications will appear.')}
                disabled={!config.notificationsEnabled}
                className="w-full h-9 bg-accent hover:bg-accent/80 hover:text-foreground text-muted-foreground border-border/50 transition-all font-medium text-xs gap-1.5"
            >
                <Play className="w-3.5 h-3.5" fill="currentColor" />
                Preview Notification
            </Button>
        </div>
    )
}
