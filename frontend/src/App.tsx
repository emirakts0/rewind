import { useState, useEffect, useCallback } from 'react'
import { Save, Square, HardDrive } from 'lucide-react'
import { api, type DisplayInfo, type EncoderInfo, type Config, type State } from '@/lib/wails'
import { formatTime, cn } from '@/lib/utils'

// Components
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Toaster } from '@/components/ui/sonner'
import { toast } from "sonner"
import { BufferSlider, BUFFER_STEPS } from '@/components/buffer-slider'
import { StatusBadge } from '@/components/status-badge'
import { ConfigPanel } from '@/components/config-panel'

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
    const [configOpen, setConfigOpen] = useState(false)

    // Helper: format buffer display (e.g. 90 -> 1:30)
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

    // Init Effect
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

                // Snap to nearest buffer step
                const nearestStep = BUFFER_STEPS.reduce((prev, curr) =>
                    Math.abs(curr - c.recordSeconds) < Math.abs(prev - c.recordSeconds) ? curr : prev
                )
                setConfig({ ...c, recordSeconds: nearestStep })
            } catch (err) {
                toast.error(`Init failed: ${err}`)
            } finally {
                setLoading(false)
            }
        }
        init()
    }, [])

    // State Polling Effect
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
            await api.setConfig(config)
            await api.start()
            setState(prev => ({ ...prev, status: 'recording' }))
            toast.success("Recording started")
        } catch (err) {
            toast.error(`${err}`)
        }
    }

    const handleStop = async () => {
        try {
            await api.stop()
            setState({ status: 'idle', bufferUsage: 0, recordingFor: 0 })
            toast.info("Recording stopped")
        } catch (err) {
            toast.error(`${err}`)
        }
    }

    const handleSave = async () => {
        try {
            const filename = await api.saveClip()
            toast.success(`Saved clip: ${filename}`, {
                description: "File saved successfully to global clips folder."
            })
        } catch (err) {
            toast.error(`${err}`)
        }
    }

    const handleSelectDirectory = useCallback(async () => {
        try {
            const path = await api.SelectDirectory()
            if (path && path.trim()) {
                setConfig(prev => ({ ...prev, outputDir: path }))
            }
        } catch (err: any) {
            console.error("Directory selection error:", err)
            toast.error("Failed to select directory")
        }
    }, [])

    const isRecording = state.status === 'recording'

    if (loading) {
        return (
            <div className="flex-1 flex items-center justify-center">
                <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin" />
            </div>
        )
    }

    return (
        <div className="h-screen w-screen bg-transparent flex flex-col overflow-hidden select-none font-sans text-foreground">
            {/* Header */}
            <header className="px-4 pt-4 pb-2 flex items-center justify-between drag-handle cursor-move z-50">
                <span className="font-extrabold text-lg tracking-tight bg-gradient-to-br from-primary via-primary/90 to-primary/70 bg-clip-text text-transparent drop-shadow-sm">
                    Rewind
                </span>
                <StatusBadge status={isRecording ? 'recording' : 'idle'} />
            </header>

            {/* Main Content Area */}
            <main className={cn(
                "flex-1 flex flex-col items-center px-8 transition-all duration-700 ease-[cubic-bezier(0.22,1,0.36,1)]",
                configOpen ? "pt-6 gap-2" : "pt-[22vh] gap-8"
            )}>
                {/* Buffer Info */}
                <div className="text-center space-y-4">
                    <div className="space-y-1">
                        <div className={cn(
                            "grid transition-all duration-700 ease-[cubic-bezier(0.22,1,0.36,1)]",
                            configOpen ? "grid-rows-[0fr] opacity-0" : "grid-rows-[1fr] opacity-100"
                        )}>
                            <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest overflow-hidden">Buffer Length</p>
                        </div>
                        <div className="flex items-baseline justify-center gap-1.5">
                            <span className={cn(
                                "font-bold tabular-nums tracking-tighter transition-all duration-700 ease-[cubic-bezier(0.22,1,0.36,1)]",
                                configOpen ? "text-5xl" : "text-7xl"
                            )}>
                                {formatBufferDisplay(config.recordSeconds)}
                            </span>
                            <span className="text-2xl text-muted-foreground font-light">
                                {getBufferUnit(config.recordSeconds)}
                            </span>
                        </div>
                    </div>
                    <Badge variant="outline" className="gap-1.5 px-3 py-1 bg-secondary/30 backdrop-blur-sm border-border/50">
                        <HardDrive className="w-3 h-3" />
                        <span className="font-normal opacity-80">Est. Memory: {estimatedMemory()}</span>
                    </Badge>
                </div>

                {/* Slider Component */}
                <div className="w-full max-w-md px-2">
                    <BufferSlider
                        value={config.recordSeconds}
                        onChange={(v) => !isRecording && setConfig(prev => ({ ...prev, recordSeconds: v }))}
                        disabled={isRecording}
                    />
                </div>

                {/* State Info */}
                <div className={cn(
                    "grid transition-all duration-700 ease-[cubic-bezier(0.22,1,0.36,1)]",
                    configOpen ? "grid-rows-[0fr] opacity-0" : "grid-rows-[1fr] opacity-100"
                )}>
                    <p className="text-xs text-muted-foreground/60 text-center h-4 overflow-hidden">
                        {isRecording
                            ? `Buffer Usage: ${state.bufferUsage}% • Recording for: ${formatTime(state.recordingFor)}`
                            : (
                                <span className="animate-pulse">Drag or select duration</span>
                            )}
                    </p>
                </div>
            </main>

            {/* Footer */}
            <footer className="p-4 pt-2 space-y-3">
                <ConfigPanel
                    open={configOpen}
                    onOpenChange={setConfigOpen}
                    config={config}
                    setConfig={setConfig}
                    displays={displays}
                    encoders={encoders}
                    disabled={isRecording}
                    onSelectDirectory={handleSelectDirectory}
                />

                <div className={cn(
                    "transition-all ease-[cubic-bezier(0.22,1,0.36,1)] transform origin-bottom",
                    configOpen
                        ? "duration-[1000ms] opacity-0 translate-y-4 scale-90 max-h-0 overflow-hidden pointer-events-none mt-0"
                        : "duration-700 opacity-100 translate-y-0 scale-100 max-h-24 mt-3"
                )}>
                    {/* Action buttons */}
                    {!isRecording ? (
                        <Button
                            onClick={handleStart}
                            size="lg"
                            className="w-full h-14 rounded-xl bg-action hover:bg-action/90 text-action-foreground text-base font-semibold shadow-lg shadow-action/20 hover:shadow-action/30 transition-all hover:scale-[1.01] active:scale-[0.99]"
                        >
                            Start
                        </Button>
                    ) : (
                        <div className="flex gap-3 animate-in fade-in slide-in-from-bottom-2 duration-300">
                            <Button
                                onClick={handleSave}
                                className="flex-[7] h-14 rounded-xl bg-action hover:bg-action/90 text-action-foreground font-semibold text-base shadow-lg shadow-action/20 hover:shadow-action/30 transition-all hover:scale-[1.01] active:scale-[0.99]"
                            >
                                <Save className="w-5 h-5 mr-2" />
                                Save Clip
                            </Button>
                            <Button
                                onClick={handleStop}
                                className="flex-[3] h-14 rounded-xl bg-destructive hover:bg-[#ff6659] text-destructive-foreground font-semibold text-base shadow-lg shadow-destructive/20 hover:shadow-destructive/30 transition-all hover:scale-[1.01] active:scale-[0.99]"
                            >
                                <Square className="w-5 h-5" />
                            </Button>
                        </div>
                    )}
                </div>
            </footer>

            <Toaster position="top-center" richColors />
        </div>
    )
}

export default App
