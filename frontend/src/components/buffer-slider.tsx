import { Slider } from '@/components/ui/slider'
import { cn } from '@/lib/utils'

export const BUFFER_STEPS = [15, 30, 60, 90, 120, 180, 300]
const BUFFER_LABELS = ['15s', '30s', '60s', '90s', '2m', '3m', '5m']

interface BufferSliderProps {
    value: number
    onChange: (v: number) => void
    disabled?: boolean
}

export function BufferSlider({
    value,
    onChange,
    disabled = false
}: BufferSliderProps) {
    const maxIndex = BUFFER_STEPS.length - 1
    const currentIndex = BUFFER_STEPS.indexOf(value)
    const SafeIndex = currentIndex !== -1 ? currentIndex : 0

    const handleValueChange = (vals: number[]) => {
        const idx = vals[0]
        onChange(BUFFER_STEPS[idx])
    }

    return (
        <div className={cn("w-full pt-4 pb-2", disabled && "opacity-50 pointer-events-none")}>
            <div className="relative mb-6">
                <Slider
                    value={[SafeIndex]}
                    min={0}
                    max={maxIndex}
                    step={1}
                    onValueChange={handleValueChange}
                    disabled={disabled}
                    className="cursor-pointer"
                />

                <div className="absolute top-1/2 -translate-y-1/2 left-0 right-0 pointer-events-none flex justify-between px-1.5">
                    {BUFFER_STEPS.map((_, i) => (
                        <div
                            key={i}
                            className={cn(
                                "w-1 h-1 rounded-full bg-foreground/20 z-0",
                                i === SafeIndex && "opacity-0"
                            )}
                        />
                    ))}
                </div>
            </div>

            <div className="flex justify-between px-0 select-none">
                {BUFFER_LABELS.map((label, i) => (
                    <div
                        key={i}
                        onClick={() => !disabled && onChange(BUFFER_STEPS[i])}
                        className={cn(
                            "text-xs cursor-pointer transition-colors w-8 text-center",
                            i === SafeIndex
                                ? "text-primary font-semibold"
                                : "text-muted-foreground hover:text-foreground"
                        )}
                    >
                        {label}
                    </div>
                ))}
            </div>
        </div>
    )
}
