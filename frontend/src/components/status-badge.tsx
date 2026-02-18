import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

interface StatusBadgeProps {
    status: string
    timer?: string
}

export function StatusBadge({ status, timer }: StatusBadgeProps) {
    const isRecording = status === 'recording'
    return (
        <Badge
            variant={isRecording ? "destructive" : "secondary"}
            className={cn(
                "gap-1.5 transition-all duration-500",
                isRecording
                    ? "bg-emerald-500/20 text-emerald-400 hover:bg-emerald-500/30 border-emerald-500/30"
                    : "bg-action/15 text-action hover:bg-action/25 border-action/20"
            )}
        >
            <div className={cn(
                "w-1.5 h-1.5 rounded-full transition-all duration-500",
                isRecording ? "bg-emerald-400 animate-pulse" : "bg-action"
            )} />
            <span className="transition-all duration-500">
                {isRecording ? 'Recording' : 'Ready'}
            </span>
            {isRecording && timer && (
                <span className="ml-1 font-mono text-emerald-300 tabular-nums">
                    {timer}
                </span>
            )}
        </Badge>
    )
}
