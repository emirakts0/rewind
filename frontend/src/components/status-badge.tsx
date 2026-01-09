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
                    ? "bg-red-500/15 text-red-500 hover:bg-red-500/25 border-red-500/20"
                    : "bg-emerald-500/15 text-emerald-500 hover:bg-emerald-500/25 border-emerald-500/20"
            )}
        >
            <div className={cn(
                "w-1.5 h-1.5 rounded-full",
                isRecording ? "bg-red-500 animate-pulse" : "bg-emerald-500"
            )} />
            {isRecording ? 'Recording' : 'Ready'}
        </Badge>
    )
}
