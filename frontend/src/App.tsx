import { useState, useEffect, useCallback, useRef } from 'react'
import {
    Circle,
    CircleDot,
    Settings2,
    ChevronUp,
    Save,
    Square,
    Monitor,
    Cpu,
    Timer,
    HardDrive,
    Check,
    X
} from 'lucide-react'
import { api, type DisplayInfo, type EncoderInfo, type Config, type State } from '@/lib/wails'
import { cn, formatTime } from '@/lib/utils'

// Fixed buffer steps (in seconds)
const BUFFER_STEPS = [15, 30, 60, 90, 120, 180, 300]
const BUFFER_LABELS = ['15s', '30s', '1m', '1.5m', '2m', '3m', '5m']

// Clean segmented slider with dots
function BufferSlider({
    value,
    onChange,
    disabled = false
}: {
    value: number
    onChange: (v: number) => void
    disabled?: boolean
}) {
    const trackRef = useRef<HTMLDivElement>(null)
    const [isDragging, setIsDragging] = useState(false)

    const currentIndex = BUFFER_STEPS.findIndex(s => s === value)
    const activeIndex = currentIndex >= 0 ? currentIndex : 0

    const getIndexFromPosition = useCallback((clientX: number) => {
        if (!trackRef.current) return activeIndex
        const rect = trackRef.current.getBoundingClientRect()
        const percent = (clientX - rect.left) / rect.width
        const rawIndex = percent * (BUFFER_STEPS.length - 1)
        return Math.max(0, Math.min(BUFFER_STEPS.length - 1, Math.round(rawIndex)))
    }, [activeIndex])

    const handleMouseDown = (e: React.MouseEvent) => {
        if (disabled) return
        setIsDragging(true)
        const newIndex = getIndexFromPosition(e.clientX)
        onChange(BUFFER_STEPS[newIndex])
    }

    useEffect(() => {
        if (!isDragging) return

        const handleMouseMove = (e: MouseEvent) => {
            const newIndex = getIndexFromPosition(e.clientX)
            if (BUFFER_STEPS[newIndex] !== value) {
                onChange(BUFFER_STEPS[newIndex])
            }
        }

        const handleMouseUp = () => setIsDragging(false)

        window.addEventListener('mousemove', handleMouseMove)
        window.addEventListener('mouseup', handleMouseUp)
        return () => {
            window.removeEventListener('mousemove', handleMouseMove)
            window.removeEventListener('mouseup', handleMouseUp)
        }
    }, [isDragging, getIndexFromPosition, onChange, value])

    return (
        <div className={cn("w-full select-none", disabled && "opacity-50 pointer-events-none")}>
            {/* Track */}
            <div
                ref={trackRef}
                className="relative h-16 cursor-pointer flex items-center"
                onMouseDown={handleMouseDown}
            >
                {/* Background track */}
                <div className="absolute inset-x-0 top-1/2 -translate-y-1/2 h-1 bg-secondary rounded-full" />

                {/* Active track */}
                <div
                    className="absolute top-1/2 -translate-y-1/2 h-1 bg-gradient-to-r from-primary to-primary/80 rounded-full transition-all duration-200"
                    style={{
                        left: 0,
                        width: `${(activeIndex / (BUFFER_STEPS.length - 1)) * 100}%`
                    }}
                />

                {/* Step dots */}
                {BUFFER_STEPS.map((_, i) => {
                    const isActive = i <= activeIndex
                    const isCurrent = i === activeIndex
                    const position = (i / (BUFFER_STEPS.length - 1)) * 100

                    return (
                        <div
                            key={i}
                            className="absolute top-1/2 -translate-y-1/2 -translate-x-1/2"
                            style={{ left: `${position}%` }}
                        >
                            <div
                                className={cn(
                                    "rounded-full transition-all duration-200",
                                    isCurrent
                                        ? "w-5 h-5 bg-primary shadow-[0_0_12px_hsl(var(--primary)/0.6)]"
                                        : isActive
                                            ? "w-3 h-3 bg-primary"
                                            : "w-2.5 h-2.5 bg-secondary border-2 border-muted-foreground/30"
                                )}
                            />
                        </div>
                    )
                })}

                {/* Thumb */}
                <div
                    className="absolute top-1/2 -translate-y-1/2 -translate-x-1/2 w-7 h-7 rounded-full bg-white shadow-lg border-2 border-primary transition-all duration-200 flex items-center justify-center"
                    style={{ left: `${(activeIndex / (BUFFER_STEPS.length - 1)) * 100}%` }}
                >
                    <div className="w-2 h-2 rounded-full bg-primary" />
                </div>
            </div>

            {/* Labels */}
            <div className="flex justify-between px-0">
                {BUFFER_LABELS.map((label, i) => (
                    <button
                        key={i}
                        onClick={() => !disabled && onChange(BUFFER_STEPS[i])}
                        className={cn(
                            "text-xs py-1 w-10 text-center transition-colors rounded",
                            i === activeIndex
                                ? "text-primary font-semibold"
                                : "text-muted-foreground hover:text-foreground"
                        )}
                    >
                        {label}
                    </button>
                ))}
            </div>
        </div>
    )
}

// Status badge
function StatusBadge({ status }: { status: string }) {
    const isRecording = status === 'recording'
    return (
        <div className={cn(
            "flex items-center gap-1.5 text-xs font-medium",
            isRecording ? "text-red-400" : "text-emerald-400"
        )}>
            <div className={cn(
                "w-2 h-2 rounded-full",
                isRecording ? "bg-red-400 animate-pulse" : "bg-emerald-400"
            )} />
            {isRecording ? 'Recording' : 'Ready'}
        </div>
    )
}

// Notification
function Notification({ message, type, onClose }: {
    message: string
    type: 'success' | 'error'
    onClose: () => void
}) {
    useEffect(() => {
        const timer = setTimeout(onClose, 3000)
        return () => clearTimeout(timer)
    }, [onClose])

    return (
        <div className={cn(
            "flex items-center gap-2 px-4 py-2.5 rounded-xl text-sm",
            type === 'success'
                ? "bg-emerald-600/20 text-emerald-400 border border-emerald-600/30"
                : "bg-red-600/20 text-red-400 border border-red-600/30"
        )}>
            {type === 'success' ? <Check className="w-4 h-4" /> : <X className="w-4 h-4" />}
            <span className="flex-1 truncate">{message}</span>
        </div>
    )
}

// Main App
function App() {
    const [displays, setDisplays] = useState<DisplayInfo[]>([])
    const [encoders, setEncoders] = useState<EncoderInfo[]>([])
    const [config, setConfig] = useState<Config>({
        displayIndex: 0,
        encoderName: '',
        fps: 60,
        bitrate: '15M',
        recordSeconds: 30,
        outputDir: './clips'
    })
    const [state, setState] = useState<State>({
        status: 'idle',
        bufferUsage: 0,
        recordingFor: 0
    })
    const [loading, setLoading] = useState(true)
    const [notification, setNotification] = useState<{ message: string; type: 'success' | 'error' } | null>(null)
    const [configOpen, setConfigOpen] = useState(false)

    const formatBufferDisplay = (seconds: number) => {
        if (seconds >= 60) {
            const mins = Math.floor(seconds / 60)
            const secs = seconds % 60
            return secs > 0 ? `${mins}:${secs.toString().padStart(2, '0')}` : `${mins}`
        }
        return seconds.toString()
    }

    const getBufferUnit = (seconds: number) => seconds >= 60 ? 'min' : 'sec'

    const estimatedMemory = useCallback(() => {
        const bitrateMap: Record<string, number> = { '8M': 8, '15M': 15, '25M': 25, '40M': 40 }
        const mbps = bitrateMap[config.bitrate] || 15
        const mb = (mbps * config.recordSeconds) / 8
        return `~${Math.round(mb)}MB`
    }, [config.bitrate, config.recordSeconds])

    useEffect(() => {
        const init = async () => {
            try {
                await api.initialize()
                const [d, e, c] = await Promise.all([
                    api.getDisplays(),
                    api.getEncoders(),
                    api.getConfig()
                ])
                setDisplays(d || [])
                setEncoders(e || [])
                const nearestStep = BUFFER_STEPS.reduce((prev, curr) =>
                    Math.abs(curr - c.recordSeconds) < Math.abs(prev - c.recordSeconds) ? curr : prev
                )
                setConfig({ ...c, recordSeconds: nearestStep })
                setLoading(false)
            } catch (err) {
                setNotification({ message: `Init failed: ${err}`, type: 'error' })
                setLoading(false)
            }
        }
        init()
    }, [])

    useEffect(() => {
        if (state.status !== 'recording') return
        const interval = setInterval(async () => {
            try {
                const s = await api.getState()
                setState(s)
            } catch (err) {
                console.error('Failed to get state:', err)
            }
        }, 500)
        return () => clearInterval(interval)
    }, [state.status])

    const handleStart = async () => {
        try {
            setNotification(null)
            await api.setConfig(config)
            await api.start()
            setState(prev => ({ ...prev, status: 'recording' }))
        } catch (err) {
            setNotification({ message: `${err}`, type: 'error' })
        }
    }

    const handleStop = async () => {
        try {
            await api.stop()
            setState({ status: 'idle', bufferUsage: 0, recordingFor: 0 })
        } catch (err) {
            setNotification({ message: `${err}`, type: 'error' })
        }
    }

    const handleSave = async () => {
        try {
            const filename = await api.saveClip()
            setNotification({ message: `Saved: ${filename}`, type: 'success' })
        } catch (err) {
            setNotification({ message: `${err}`, type: 'error' })
        }
    }

    const isRecording = state.status === 'recording'

    if (loading) {
        return (
            <div className="flex-1 flex items-center justify-center">
                <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin" />
            </div>
        )
    }

    return (
        <div className="flex-1 flex flex-col w-full">
            {/* Header */}
            <header className="flex items-center justify-between p-5 pb-3">
                <div className="flex items-center gap-2">
                    <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-primary to-primary/60 flex items-center justify-center">
                        <Circle className="w-4 h-4" />
                    </div>
                    <span className="font-semibold text-lg">Rewind</span>
                </div>
                <StatusBadge status={state.status} />
            </header>

            {/* Main Content */}
            <main className="flex-1 flex flex-col items-center justify-center px-6 gap-6">
                {/* Buffer Display */}
                <div className="text-center space-y-2">
                    <p className="text-xs text-muted-foreground uppercase tracking-wider">Buffer Length</p>
                    <div className="flex items-baseline justify-center gap-1">
                        <span className="text-6xl font-bold tabular-nums">
                            {formatBufferDisplay(config.recordSeconds)}
                        </span>
                        <span className="text-xl text-muted-foreground">
                            {getBufferUnit(config.recordSeconds)}
                        </span>
                    </div>
                    <div className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-secondary/50 text-xs text-muted-foreground">
                        <HardDrive className="w-3 h-3" />
                        Memory Usage: {estimatedMemory()}
                    </div>
                </div>

                {/* Buffer Slider */}
                <div className="w-full px-2">
                    <BufferSlider
                        value={config.recordSeconds}
                        onChange={(v) => setConfig(prev => ({ ...prev, recordSeconds: v }))}
                        disabled={isRecording}
                    />
                </div>

                {/* Helper Text */}
                <p className="text-xs text-muted-foreground text-center">
                    {isRecording
                        ? `Recording: ${formatTime(state.recordingFor)} | Buffer: ${state.bufferUsage}%`
                        : 'Drag or tap to select buffer duration.'}
                </p>
            </main>

            {/* Footer Controls */}
            <footer className="p-4 pt-2 space-y-3">
                {/* Advanced Configuration */}
                <div className="rounded-xl border border-border overflow-hidden">
                    <button
                        onClick={() => setConfigOpen(!configOpen)}
                        disabled={isRecording}
                        className={cn(
                            "w-full px-4 py-3 flex items-center justify-between text-sm",
                            "hover:bg-secondary/50 transition-colors",
                            isRecording && "opacity-50 pointer-events-none"
                        )}
                    >
                        <div className="flex items-center gap-2">
                            <Settings2 className="w-4 h-4 text-muted-foreground" />
                            <span>Advanced Configuration</span>
                        </div>
                        <ChevronUp className={cn(
                            "w-4 h-4 text-muted-foreground transition-transform duration-300",
                            configOpen ? "rotate-180" : "rotate-0"
                        )} />
                    </button>

                    <div className={cn(
                        "grid transition-all duration-300 ease-in-out",
                        configOpen ? "grid-rows-[1fr] opacity-100" : "grid-rows-[0fr] opacity-0"
                    )}>
                        <div className="overflow-hidden">
                            <div className="px-4 pb-4 pt-2 space-y-3 border-t border-border">
                                <div className="space-y-1.5">
                                    <label className="text-xs text-muted-foreground flex items-center gap-1.5">
                                        <Monitor className="w-3 h-3" /> Display
                                    </label>
                                    <select
                                        value={config.displayIndex}
                                        onChange={e => setConfig(prev => ({ ...prev, displayIndex: parseInt(e.target.value) }))}
                                        className="w-full h-9 px-3 rounded-lg bg-secondary border-0 text-sm focus:ring-2 focus:ring-ring outline-none"
                                    >
                                        {displays.map(d => (
                                            <option key={d.index} value={d.index}>
                                                {d.name || `Display ${d.index + 1}`} ({d.width}x{d.height}){d.isPrimary ? ' ★' : ''}
                                            </option>
                                        ))}
                                    </select>
                                </div>

                                <div className="space-y-1.5">
                                    <label className="text-xs text-muted-foreground flex items-center gap-1.5">
                                        <Cpu className="w-3 h-3" /> Encoder
                                    </label>
                                    <select
                                        value={config.encoderName}
                                        onChange={e => setConfig(prev => ({ ...prev, encoderName: e.target.value }))}
                                        className="w-full h-9 px-3 rounded-lg bg-secondary border-0 text-sm focus:ring-2 focus:ring-ring outline-none"
                                    >
                                        {encoders.map(e => (
                                            <option key={e.name} value={e.name}>
                                                {e.name} ({e.gpuName})
                                            </option>
                                        ))}
                                    </select>
                                </div>

                                <div className="grid grid-cols-2 gap-3">
                                    <div className="space-y-1.5">
                                        <label className="text-xs text-muted-foreground flex items-center gap-1.5">
                                            <Timer className="w-3 h-3" /> FPS
                                        </label>
                                        <select
                                            value={config.fps}
                                            onChange={e => setConfig(prev => ({ ...prev, fps: parseInt(e.target.value) }))}
                                            className="w-full h-9 px-3 rounded-lg bg-secondary border-0 text-sm focus:ring-2 focus:ring-ring outline-none"
                                        >
                                            <option value={30}>30</option>
                                            <option value={60}>60</option>
                                            <option value={120}>120</option>
                                        </select>
                                    </div>
                                    <div className="space-y-1.5">
                                        <label className="text-xs text-muted-foreground">Quality</label>
                                        <select
                                            value={config.bitrate}
                                            onChange={e => setConfig(prev => ({ ...prev, bitrate: e.target.value }))}
                                            className="w-full h-9 px-3 rounded-lg bg-secondary border-0 text-sm focus:ring-2 focus:ring-ring outline-none"
                                        >
                                            <option value="8M">Medium</option>
                                            <option value="15M">High</option>
                                            <option value="25M">Ultra</option>
                                            <option value="40M">Extreme</option>
                                        </select>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                {/* Notification */}
                {notification && (
                    <Notification
                        message={notification.message}
                        type={notification.type}
                        onClose={() => setNotification(null)}
                    />
                )}

                {/* Action buttons */}
                {!isRecording ? (
                    <button
                        onClick={handleStart}
                        className="w-full h-14 rounded-2xl bg-primary text-primary-foreground font-semibold flex items-center justify-center gap-2 hover:bg-primary/90 transition-colors"
                    >
                        <CircleDot className="w-5 h-5" />
                        Start Buffering
                    </button>
                ) : (
                    <div className="flex gap-3">
                        <button
                            onClick={handleSave}
                            className="flex-[7] h-12 rounded-xl bg-emerald-600 text-white font-medium flex items-center justify-center gap-2 hover:bg-emerald-700 transition-colors"
                        >
                            <Save className="w-4 h-4" />
                            Save Clip
                        </button>
                        <button
                            onClick={handleStop}
                            className="flex-[3] h-12 rounded-xl bg-secondary text-secondary-foreground font-medium flex items-center justify-center gap-2 hover:bg-secondary/80 transition-colors"
                        >
                            <Square className="w-4 h-4" />
                        </button>
                    </div>
                )}
            </footer>
        </div>
    )
}

export default App
