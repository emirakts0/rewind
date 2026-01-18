import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

export function StatusBadge({ status }: { status: string }) {
    const isRecording = status === 'recording'
    return (
        <Badge
            variant={isRecording ? "destructive" : "secondary"}
            className={cn(
                "gap-1.5",
                isRecording
                    ? "bg-emerald-500/20 text-emerald-400 hover:bg-emerald-500/30 border-emerald-500/30"
                    : "bg-action/15 text-action hover:bg-action/25 border-action/20"
            )}
        >
            <div className={cn(
                "w-1.5 h-1.5 rounded-full",
                isRecording ? "bg-emerald-400 animate-pulse" : "bg-action"
            )} />
            {isRecording ? 'Recording' : 'Ready'}
        </Badge>
    )
}
