import { useState, useEffect, useCallback } from 'react'
import { Save, Square, HardDrive } from 'lucide-react'
import { api, type DisplayInfo, type EncoderInfo, type Config, type State } from '@/lib/wails'
import { formatTime, formatBufferDisplay, getBufferUnit, formatError, cn } from '@/lib/utils'

// Components
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Toaster } from '@/components/ui/sonner'
import { toast } from "sonner"
import { BufferSlider, BUFFER_STEPS } from '@/components/buffer-slider'
import { StatusBadge } from '@/components/status-badge'
import { ConfigPanel } from '@/components/config-panel'
import { ClipsDrawer } from '@/components/clips-drawer'
import { TitleBar } from '@/components/title-bar'
import { Kbd, KbdGroup } from '@/components/ui/kbd'

function App() {
    const [displays, setDisplays] = useState<DisplayInfo[]>([])
    const [encoders, setEncoders] = useState<EncoderInfo[]>([])
    const [inputDevices, setInputDevices] = useState<string[]>([])
    const [outputDevices, setOutputDevices] = useState<string[]>([])
    const [config, setConfig] = useState<Config>({
        displayIndex: 0,
        encoderName: '',
        fps: 30,
        bitrate: '15M',
        recordSeconds: 30,
        outputDir: './clips',
        convertToMP4: true,
        microphoneDevice: '',
        micVolume: 100,
        systemAudioDevice: '',
        sysVolume: 100,
    })
    const [state, setState] = useState<State>({
        status: 'idle',
        bufferUsage: 0,
        recordingFor: 0
    })
    const [loading, setLoading] = useState(true)
    const [configOpen, setConfigOpen] = useState(false)
    const [estimatedMemory, setEstimatedMemory] = useState("~0MB")

    useEffect(() => {
        api.estimateMemory(
            config.bitrate,
            config.recordSeconds,
            config.microphoneDevice !== '',
            config.systemAudioDevice !== ''
        )
            .then(setEstimatedMemory)
            .catch(err => console.error(err))
    }, [config.bitrate, config.recordSeconds, config.microphoneDevice, config.systemAudioDevice])

    // Update encoders when display changes
    useEffect(() => {
        if (loading) return
        api.getEncodersForDisplay(config.displayIndex)
            .then(setEncoders)
            .catch(err => console.error("Failed to update encoders:", err))
    }, [config.displayIndex, loading])

    // Init Effect
    useEffect(() => {
        const init = async () => {
            try {
                await api.initialize()
                const [d, e, inputs, outputs, c, s] = await Promise.all([
                    api.getDisplays(),
                    api.getEncoders(),
                    api.getInputDevices(),
                    api.getOutputDevices(),
                    api.getConfig(),
                    api.getState()
                ])
                setDisplays(d || [])
                setEncoders(e || [])
                setInputDevices(inputs || [])
                setOutputDevices(outputs || [])

                // Snap to nearest buffer step
                const nearestStep = BUFFER_STEPS.reduce((prev, curr) =>
                    Math.abs(curr - c.recordSeconds) < Math.abs(prev - c.recordSeconds) ? curr : prev
                )
                setConfig({ ...c, recordSeconds: nearestStep })

                // Set initial state from Go backend (important when window reopens)
                if (s) {
                    setState(s)
                }
            } catch (err: any) {
                toast.error(`Init failed: ${formatError(err)}`)
            } finally {
                setLoading(false)
            }
        }
        init()
    }, [])

    // State Management Effect (Events + Polling)
    useEffect(() => {
        // Listen for state changes from backend (Tray, Shortcuts)
        const unsub = api.Events.On('state-changed', (event: any) => {
            const s = event.data as State
            console.log("State changed:", s)
            setState(s)

            // Auto-close config panel when recording starts (e.g. via shortcut)
            if (s.status === 'recording' && configOpen) {
                setConfigOpen(false)
            }
        })

        // Poll for buffer usage when recording
        let interval: NodeJS.Timeout
        if (state.status === 'recording') {
            interval = setInterval(async () => {
                try {
                    const s = await api.getState()
                    setState(s)
                } catch (err) {
                    console.error('Failed to get state:', err)
                }
            }, 500)
        }

        return () => {
            unsub()
            if (interval) clearInterval(interval)
        }
    }, [state.status, configOpen])

    const handleStart = async () => {
        try {
            await api.setConfig(config)
            await api.start()
            setState(prev => ({ ...prev, status: 'recording' }))
            toast.success("Recording started")
        } catch (err: any) {
            toast.error(formatError(err))
        }
    }

    const handleStop = async () => {
        try {
            await api.stop()
            setState({ status: 'idle', bufferUsage: 0, recordingFor: 0 })
            toast.info("Recording stopped")
        } catch (err: any) {
            toast.error(formatError(err))
        }
    }

    const handleSave = async () => {
        try {
            const filename = await api.saveClip()

            let title = `Saved clip: ${filename}`
            let description = "File saved successfully to global clips folder."

            if (!config.convertToMP4) {
                title = "Clip saved"
                description = `Raw data saved to: ${filename}`
            }

            toast.success(title, {
                description: description
            })
        } catch (err: any) {
            toast.error(formatError(err))
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
            toast.error(formatError(err))
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
        <div className="h-screen w-screen bg-transparent flex flex-col overflow-hidden select-none font-sans text-foreground relative">

            {/* Custom Title Bar */}
            <TitleBar>
                <StatusBadge status={isRecording ? 'recording' : 'idle'} />
                <ClipsDrawer />
            </TitleBar>

            {/* Main Content Area */}
            <main className={cn(
                "flex-1 flex flex-col items-center px-8 transition-all duration-700 ease-[cubic-bezier(0.22,1,0.36,1)]",
                configOpen ? "pt-6 gap-2" : "pt-[14vh] gap-5"
            )}>
                {/* Buffer Info */}
                <div className="text-center space-y-3">
                    <div className="space-y-1">
                        <div className={cn(
                            "grid transition-all duration-700 ease-[cubic-bezier(0.22,1,0.36,1)]",
                            configOpen ? "grid-rows-[0fr] opacity-0" : "grid-rows-[1fr] opacity-100"
                        )}>
                            <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest overflow-hidden">Replay Buffer</p>
                        </div>
                        <div className="flex items-baseline justify-center gap-2">
                            <span className={cn(
                                "font-bold tabular-nums tracking-tighter transition-all duration-700 ease-[cubic-bezier(0.22,1,0.36,1)]",
                                configOpen ? "text-6xl" : "text-8xl"
                            )}>
                                {formatBufferDisplay(config.recordSeconds)}
                            </span>
                            <span className="text-3xl text-muted-foreground font-light">
                                {getBufferUnit(config.recordSeconds)}
                            </span>
                        </div>
                    </div>
                    <Badge variant="outline" className="gap-1.5 px-3 py-1 bg-secondary/30 backdrop-blur-sm border-border/50">
                        <HardDrive className="w-3 h-3" />
                        <span className="font-normal opacity-80">Est. Memory: {estimatedMemory}</span>
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
                    <div className="flex flex-col items-center gap-3 overflow-hidden">
                        <p className="text-xs text-muted-foreground/60 text-center h-4">
                            {isRecording
                                ? `Buffer Usage: ${state.bufferUsage}% • Recording for: ${formatTime(state.recordingFor)}`
                                : " "
                            }
                        </p>

                        <div className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 mt-2 opacity-60 items-center">
                            <span className="text-[10px] text-muted-foreground font-medium uppercase tracking-wider text-left">Start/Stop</span>
                            <KbdGroup>
                                <Kbd>Ctrl</Kbd>
                                <span>+</span>
                                <Kbd>F9</Kbd>
                            </KbdGroup>

                            <span className="text-[10px] text-muted-foreground font-medium uppercase tracking-wider text-left">Save Clip</span>
                            <KbdGroup>
                                <Kbd>Ctrl</Kbd>
                                <span>+</span>
                                <Kbd>F10</Kbd>
                            </KbdGroup>
                        </div>
                    </div>
                </div>
            </main>

            {/* Footer */}
            <footer className="p-4 pt-2 space-y-3 relative z-10">
                <ConfigPanel
                    open={configOpen}
                    onOpenChange={setConfigOpen}
                    config={config}
                    setConfig={setConfig}
                    displays={displays}
                    encoders={encoders}
                    inputDevices={inputDevices}
                    outputDevices={outputDevices}
                    disabled={isRecording}
                    onSelectDirectory={handleSelectDirectory}
                />

                <div className={cn(
                    "transition-all ease-[cubic-bezier(0.22,1,0.36,1)] transform origin-bottom",
                    configOpen
                        ? "duration-1000 opacity-0 translate-y-4 scale-90 max-h-0 overflow-hidden pointer-events-none mt-0"
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
                                className="flex-[3] h-14 rounded-xl bg-destructive hover:bg-destructive text-destructive-foreground font-semibold text-base shadow-lg shadow-destructive/20 transition-all hover:scale-[1.01] active:scale-[0.99]"
                            >
                                <Square className="w-5 h-5" />
                            </Button>
                        </div>
                    )}
                </div>
            </footer>

            <Toaster position="top-center" />
        </div>
    )
}

export default App
