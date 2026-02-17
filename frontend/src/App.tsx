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
        outputDir: './clips',
        microphoneDevice: -1,
        systemAudioDevice: -1,
        showCursor: true,
        showBorder: false,
    })
    const [state, setState] = useState<State>({
        status: 'idle',
        bufferUsage: 0,
        recordingFor: 0,
        diskUsageMB: 0,
        memoryUsageMB: 0,
        estimate: {
            diskMB: 0,
            memoryMB: 0,
            totalMB: 0
        }
    })
    const [loading, setLoading] = useState(true)
    const [configOpen, setConfigOpen] = useState(false)
    const [isSaving, setIsSaving] = useState(false)
    const [saveDisabledUntil, setSaveDisabledUntil] = useState<number>(0)
    
    const isRecording = state.status === 'recording'
    
    const calculateEstimate = (cfg: Config) => {
        const diskMB = (cfg.bitrate * cfg.recordSeconds) / 8
        const memoryMB = 0
        return { diskMB, memoryMB, totalMB: diskMB + memoryMB }
    }
    
    const currentEstimate = isRecording ? state.estimate : calculateEstimate(config)
    
    const estimateDiskDisplay = currentEstimate.diskMB > 0 
        ? `${currentEstimate.diskMB.toFixed(0)}MB` 
        : "0MB"
    
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
                setConfig({ ...c, recordSeconds: nearestStep })

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
        const unsub = api.Events.On('state-changed', (event: any) => {
            const s = event.data as State
            console.log("State changed:", s)
            setState(s)

            if (s.status === 'recording' && configOpen) {
                setConfigOpen(false)
            }
        })

        return () => {
            unsub()
        }
    }, [configOpen])

    const handleStart = async () => {
        try {
            await api.setConfig(config)
            await api.start()
            toast.success("Recording started")
        } catch (err: any) {
            toast.error(formatError(err))
        }
    }

    const handleStop = async () => {
        try {
            await api.stop()
            toast.info("Recording stopped")
        } catch (err: any) {
            toast.error(formatError(err))
        }
    }

    const handleSave = async () => {
        if (isSaving) return
        
        const now = Date.now()
        if (now < saveDisabledUntil) {
            const remaining = Math.ceil((saveDisabledUntil - now) / 1000)
            toast.error(`Please wait ${remaining} seconds before saving another clip`)
            return
        }

        setIsSaving(true)
        try {
            const filename = await api.saveClip()
            toast.success(`Saved: ${filename}`, {
                description: "Clip saved successfully"
            })
            setSaveDisabledUntil(Date.now() + 5000)
        } catch (err: any) {
            toast.error(formatError(err))
        } finally {
            setIsSaving(false)
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
                <StatusBadge status={isRecording ? 'recording' : 'idle'} />
                <ClipsDrawer />
            </TitleBar>

            <main className={cn(
                "flex-1 flex flex-col items-center px-8 transition-all duration-700 ease-[cubic-bezier(0.22,1,0.36,1)]",
                configOpen ? "pt-6 gap-2" : "pt-[14vh] gap-5"
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

                <div className="w-full max-w-md px-2">
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
                        <p className="text-xs text-muted-foreground/60 text-center h-4">
                            {isRecording ? formatTime(state.recordingFor) : " "}
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
                            size="lg"
                            className="w-full h-14 rounded-xl bg-action hover:bg-action/90 text-action-foreground text-base font-semibold shadow-lg shadow-action/20 hover:shadow-action/30 transition-all hover:scale-[1.01] active:scale-[0.99]"
                        >
                            Start
                        </Button>
                    ) : (
                        <div className="flex gap-3 animate-in fade-in slide-in-from-bottom-2 duration-300">
                            <Button
                                onClick={handleSave}
                                disabled={isSaving || Date.now() < saveDisabledUntil}
                                className="flex-[7] h-14 rounded-xl bg-action hover:bg-action/90 text-action-foreground font-semibold text-base shadow-lg shadow-action/20 hover:shadow-action/30 transition-all hover:scale-[1.01] active:scale-[0.99] disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:scale-100"
                            >
                                <Save className="w-5 h-5 mr-2" />
                                {isSaving ? 'Saving...' : 'Save Clip'}
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
