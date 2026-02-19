import { useState, useEffect, useCallback } from 'react'
import { Save, Square, HardDrive } from 'lucide-react'
import { api, type DisplayInfo, type Config, type State } from '@/lib/wails'
import { formatTime, formatBufferDisplay, getBufferUnit, formatError, cn } from '@/lib/utils'

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
    const [inputDevices, setInputDevices] = useState<string[]>([])
    const [outputDevices, setOutputDevices] = useState<string[]>([])
    const [config, setConfig] = useState<Config>({
        displayIndex: 0,
        fps: 30,
        bitrate: 15,
        recordSeconds: 30,
        segmentDurationSec: 5,
        outputDir: './clips',
        microphoneDevice: -1,
        systemAudioDevice: -1,
        microphoneVolume: 100,
        systemAudioVolume: 100,
        showCursor: true,
        showBorder: false,
    })
    const [state, setState] = useState<State>({
        status: 'idle',
        bufferUsage: 0,
        recordingFor: 0,
        diskUsageMB: 0,
        memoryUsageMB: 0
    })
    const [loading, setLoading] = useState(true)
    const [configOpen, setConfigOpen] = useState(false)
    
    const isRecording = state.status === 'recording' || state.status === 'saving'
    
    // Calculate estimate based on current config (for UI updates before recording starts)
    const calculateEstimate = (cfg: Config) => {
        const actualBufferSeconds = cfg.recordSeconds + cfg.segmentDurationSec
        const diskMB = (cfg.bitrate * actualBufferSeconds) / 8
        return diskMB
    }
    
    const estimateDiskDisplay = isRecording
        ? `${state.diskUsageMB.toFixed(1)}MB`
        : `${calculateEstimate(config).toFixed(0)}MB`
    
    const actualDiskDisplay = state.diskUsageMB > 0 
        ? `${state.diskUsageMB.toFixed(1)}MB` 
        : "0MB"
    const actualMemoryDisplay = state.memoryUsageMB > 0 
        ? `${state.memoryUsageMB.toFixed(1)}MB` 
        : "0MB"

    useEffect(() => {
        const init = async () => {
            try {
                const [d, inputs, outputs, c, s] = await Promise.all([
                    api.getDisplays(),
                    api.getInputDevices(),
                    api.getOutputDevices(),
                    api.getConfig(),
                    api.getState()
                ])
                setDisplays(d || [])
                setInputDevices(inputs || [])
                setOutputDevices(outputs || [])

                const nearestStep = BUFFER_STEPS.reduce((prev, curr) =>
                    Math.abs(curr - c.recordSeconds) < Math.abs(prev - c.recordSeconds) ? curr : prev
                )
                setConfig({ 
                    ...c, 
                    recordSeconds: nearestStep,
                    microphoneVolume: c.microphoneVolume ?? 100,
                    systemAudioVolume: c.systemAudioVolume ?? 100,
                })

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

    useEffect(() => {
        const unsubState = api.Events.On('state-changed', (event: any) => {
            const s = event.data as State
            console.log("State changed:", s)
            setState(s)

            if (s.status === 'recording' && configOpen) {
                setConfigOpen(false)
            }
        })

        const unsubRuntimeError = api.Events.On('runtime-error', (event: any) => {
            const { level, message } = event.data as { level: string; message: string }
            console.log("Runtime error:", level, message)
            
            if (level === 'error') {
                toast.error(message, {
                    description: "Recording error",
                    duration: 5000,
                })
            } else if (level === 'warning') {
                toast.warning(message, {
                    description: "Recording warning",
                    duration: 4000,
                })
            }
        })

        return () => {
            unsubState()
            unsubRuntimeError()
        }
    }, [configOpen])

    const handleStart = async () => {
        try {
            await api.setConfig(config)
            await api.start()
            toast.success("Recording started")
        } catch (err: any) {
            const errorMsg = formatError(err)
            if (errorMsg.includes("already recording")) {
                toast.warning("Already recording")
            } else if (errorMsg.includes("cannot start while saving")) {
                toast.warning("Please wait for the current save to complete")
            } else {
                toast.error(errorMsg)
            }
        }
    }

    const handleStop = async () => {
        try {
            await api.stop()
            toast.info("Recording stopped")
        } catch (err: any) {
            const errorMsg = formatError(err)
            if (errorMsg.includes("not recording")) {
                toast.warning("Not currently recording")
            } else if (errorMsg.includes("cannot stop while saving")) {
                toast.warning("Please wait for the current save to complete")
            } else {
                toast.error(errorMsg)
            }
        }
    }

    const handleSave = async () => {
        if (state.status === 'saving') return

        try {
            const filename = await api.saveClip()
            toast.success(`Saved: ${filename}`, {
                description: "Clip saved successfully"
            })
        } catch (err: any) {
            const errorMsg = formatError(err)
            if (errorMsg.includes("not recording")) {
                toast.warning("Not currently recording")
            } else if (errorMsg.includes("save already in progress")) {
                toast.warning("Save already in progress")
            } else if (errorMsg.includes("please wait") && errorMsg.includes("seconds")) {
                toast.warning(errorMsg)
            } else if (errorMsg.includes("timed out")) {
                toast.error("Save operation timed out (45s limit)", {
                    description: "Try reducing buffer size or closing other applications"
                })
            } else {
                toast.error(errorMsg)
            }
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

    const handleRefreshDevices = useCallback(async () => {
        try {
            await api.refreshConfig()

            const [displays, inputs, outputs, cfg] = await Promise.all([
                api.getDisplays(),
                api.getInputDevices(),
                api.getOutputDevices(),
                api.getConfig()
            ])
            
            setDisplays(displays || [])
            setInputDevices(inputs || [])
            setOutputDevices(outputs || [])
            setConfig(prev => ({ ...prev, ...cfg }))
            
            toast.success("Config refreshed")
        } catch (err: any) {
            console.error("Config refresh error:", err)
            toast.error(formatError(err))
        }
    }, [])

    if (loading) {
        return (
            <div className="flex-1 flex items-center justify-center">
                <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin" />
            </div>
        )
    }

    return (
        <div className="h-screen w-screen bg-transparent flex flex-col overflow-hidden select-none font-sans text-foreground relative">
            <TitleBar>
                <StatusBadge 
                    status={isRecording ? 'recording' : 'idle'} 
                    timer={isRecording ? formatTime(state.recordingFor) : undefined}
                />
                <ClipsDrawer />
            </TitleBar>

            <main className={cn(
                "flex-1 flex flex-col items-center px-8 transition-all duration-700 ease-[cubic-bezier(0.22,1,0.36,1)]",
                configOpen ? "pt-6 gap-2" : isRecording ? "pt-[14vh] gap-2" : "pt-[14vh] gap-5"
            )}>
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
                        <span className="font-normal opacity-80">
                            {isRecording 
                                ? `Disk: ${actualDiskDisplay} • RAM: ${actualMemoryDisplay}` 
                                : `Est. Disk: ${estimateDiskDisplay}`
                            }
                        </span>
                    </Badge>
                </div>

                <div className={cn(
                    "w-full max-w-md px-2 transition-all duration-700 ease-[cubic-bezier(0.22,1,0.36,1)]",
                    isRecording ? "opacity-0 scale-95 max-h-0 mb-0" : "opacity-100 scale-100 max-h-24 mb-0"
                )}>
                    <BufferSlider
                        value={config.recordSeconds}
                        onChange={(v) => !isRecording && setConfig(prev => ({ ...prev, recordSeconds: v }))}
                        disabled={isRecording}
                    />
                </div>

                <div className={cn(
                    "grid transition-all duration-700 ease-[cubic-bezier(0.22,1,0.36,1)]",
                    configOpen ? "grid-rows-[0fr] opacity-0" : "grid-rows-[1fr] opacity-100"
                )}>
                    <div className="flex flex-col items-center gap-3 overflow-hidden">
                        <div className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 mt-4 opacity-60 items-center">
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

            <footer className="p-4 pt-2 space-y-3 relative z-10">
                <ConfigPanel
                    open={configOpen}
                    onOpenChange={setConfigOpen}
                    config={config}
                    setConfig={setConfig}
                    displays={displays}
                    inputDevices={inputDevices}
                    outputDevices={outputDevices}
                    disabled={isRecording}
                    onSelectDirectory={handleSelectDirectory}
                    onRefreshDevices={handleRefreshDevices}
                />

                <div className={cn(
                    "transition-all ease-[cubic-bezier(0.22,1,0.36,1)] transform origin-bottom",
                    configOpen
                        ? "duration-1000 opacity-0 translate-y-4 scale-90 max-h-0 overflow-hidden pointer-events-none mt-0"
                        : "duration-700 opacity-100 translate-y-0 scale-100 max-h-24 mt-3"
                )}>
                    {!isRecording ? (
                        <Button
                            onClick={handleStart}
                            disabled={state.status === 'saving'}
                            size="lg"
                            className="w-full h-14 rounded-xl bg-action hover:bg-action/90 text-action-foreground text-base font-semibold shadow-lg shadow-action/20 hover:shadow-action/30 transition-all hover:scale-[1.01] active:scale-[0.99] disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:scale-100"
                        >
                            {state.status === 'saving' ? 'Saving...' : 'Start'}
                        </Button>
                    ) : (
                        <div className="flex gap-3 animate-in fade-in slide-in-from-bottom-2 duration-300">
                            <Button
                                onClick={handleSave}
                                disabled={state.status === 'saving'}
                                className="flex-[7] h-14 rounded-xl bg-action hover:bg-action/90 text-action-foreground font-semibold text-base shadow-lg shadow-action/20 hover:shadow-action/30 transition-all hover:scale-[1.01] active:scale-[0.99] disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:scale-100"
                            >
                                <Save className="w-5 h-5 mr-2" />
                                {state.status === 'saving' ? 'Saving...' : 'Save Clip'}
                            </Button>
                            <Button
                                onClick={handleStop}
                                disabled={state.status === 'saving'}
                                className="flex-[3] h-14 rounded-xl bg-destructive hover:bg-destructive text-destructive-foreground font-semibold text-base shadow-lg shadow-destructive/20 transition-all hover:scale-[1.01] active:scale-[0.99] disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:scale-100"
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
