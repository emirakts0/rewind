import { Settings2, ChevronUp, Monitor, Cpu, Timer, Sparkles, Folder, Mic, Info, Volume, Volume1, Volume2, VolumeX, SlidersHorizontal } from 'lucide-react'
import { Switch } from "@/components/ui/switch"
import {
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger,
} from "@/components/ui/tooltip"
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'
import {
    Collapsible,
    CollapsibleContent,
    CollapsibleTrigger
} from '@/components/ui/collapsible'
import {
    Tabs,
    TabsContent,
    TabsList,
    TabsTrigger,
} from "@/components/ui/tabs"
import { Slider } from "@/components/ui/slider"
import { cn } from '@/lib/utils'
import type { Config, DisplayInfo, EncoderInfo } from '@/lib/wails'
import { ScrollArea } from "@/components/ui/scroll-area"
import { useMemo, useState } from 'react'

interface ConfigPanelProps {
    open: boolean
    onOpenChange: (open: boolean) => void
    config: Config
    setConfig: React.Dispatch<React.SetStateAction<Config>>
    displays: DisplayInfo[]
    encoders: EncoderInfo[]
    inputDevices: string[]
    outputDevices: string[]
    disabled?: boolean
    onSelectDirectory: () => void
}

// Standard FPS options to include if below display Hz
const STANDARD_FPS = [60, 30, 24]

export function ConfigPanel({
    open,
    onOpenChange,
    config,
    setConfig,
    displays,
    encoders,
    inputDevices,
    outputDevices,
    disabled,
    onSelectDirectory
}: ConfigPanelProps) {
    // Volume control visibility states
    const [showMicVolume, setShowMicVolume] = useState(false)
    const [showSysVolume, setShowSysVolume] = useState(false)

    // Get current display's refresh rate
    const selectedDisplay = displays.find(d => d.index === config.displayIndex)
    const maxHz = selectedDisplay?.refreshRate || 60

    // Build FPS options: native Hz + standard options below it
    const fpsOptions = useMemo(() => {
        const options = new Set<number>()
        options.add(maxHz) // Always include native Hz
        STANDARD_FPS.forEach(fps => {
            if (fps <= maxHz) options.add(fps)
        })
        return Array.from(options).sort((a, b) => b - a) // Descending
    }, [maxHz])

    // Ensure current FPS is valid for selected display
    useMemo(() => {
        if (config.fps > maxHz) {
            setConfig(prev => ({ ...prev, fps: maxHz }))
        }
    }, [maxHz, config.fps, setConfig])

    return (
        <Collapsible open={open} onOpenChange={onOpenChange}>
            <Card className={cn("border-border/50 shadow-sm transition-all duration-300", disabled && "opacity-50 pointer-events-none")}>
                <CollapsibleTrigger asChild disabled={disabled}>
                    <Button
                        variant="ghost"
                        className="w-full px-4 py-3 h-auto flex items-center justify-between hover:bg-transparent hover:text-foreground"
                    >
                        <div className="flex items-center gap-2.5">
                            <Settings2 className="w-4 h-4 text-muted-foreground" />
                            <span className="font-medium text-sm">Configuration</span>
                        </div>
                        <ChevronUp className={cn(
                            "w-4 h-4 text-muted-foreground/70 transition-transform duration-300",
                            open ? "rotate-180" : "rotate-0"
                        )} />
                    </Button>
                </CollapsibleTrigger>

                <CollapsibleContent>
                    <CardContent className="px-4 pb-0 pt-4 border-t border-border/40">
                        {/* Output Directory */}
                        <div className="space-y-2 mb-4">
                            <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider flex items-center gap-1.5">
                                <Folder className="w-3 h-3" /> Output Folder
                            </label>
                            <Button
                                variant="outline"
                                size="sm"
                                onClick={onSelectDirectory}
                                className="w-full justify-start text-left font-normal bg-accent border-border/50 h-9 truncate px-3 text-muted-foreground hover:text-foreground hover:bg-accent/80"
                                title={config.outputDir}
                            >
                                {config.outputDir || "./clips"}
                            </Button>
                        </div>

                        <Tabs defaultValue="video" className="w-full">
                            <TabsList className="grid w-full grid-cols-2 mb-2">
                                <TabsTrigger value="video">Video</TabsTrigger>
                                <TabsTrigger value="audio">Audio</TabsTrigger>
                            </TabsList>

                            <TabsContent value="video" className="space-y-4 animate-in slide-in-from-left-2 duration-300 fade-in-0 mt-0">
                                <ScrollArea className="h-[280px] -mx-4 w-[calc(100%+2rem)]">
                                    <div className="space-y-3 px-4 py-2">

                                        {/* Display Select */}
                                        <div className="space-y-1.5">
                                            <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider flex items-center gap-1.5">
                                                <Monitor className="w-3 h-3" /> Display
                                            </label>
                                            <Select
                                                value={config.displayIndex.toString()}
                                                onValueChange={(v) => setConfig(prev => ({ ...prev, displayIndex: parseInt(v) }))}
                                            >
                                                <SelectTrigger className="h-9 bg-accent border-border/50 focus:ring-1 focus:ring-primary/20">
                                                    <SelectValue placeholder="Select display" />
                                                </SelectTrigger>
                                                <SelectContent>
                                                    {displays.map(d => (
                                                        <SelectItem key={d.index} value={d.index.toString()}>
                                                            {d.name || `Display ${d.index + 1}`} ({d.width}x{d.height}){d.isPrimary ? ' ★' : ''}
                                                        </SelectItem>
                                                    ))}
                                                </SelectContent>
                                            </Select>
                                        </div>

                                        {/* Encoder Select */}
                                        <div className="space-y-1.5">
                                            <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider flex items-center gap-1.5">
                                                <Cpu className="w-3 h-3" /> Encoder
                                            </label>
                                            <Select
                                                value={config.encoderName}
                                                onValueChange={(v) => setConfig(prev => ({ ...prev, encoderName: v }))}
                                            >
                                                <SelectTrigger className="h-9 bg-accent border-border/50">
                                                    <SelectValue placeholder="Select encoder" />
                                                </SelectTrigger>
                                                <SelectContent>
                                                    {encoders.map(e => (
                                                        <SelectItem key={e.name} value={e.name}>
                                                            {e.name} ({e.gpuName})
                                                        </SelectItem>
                                                    ))}
                                                </SelectContent>
                                            </Select>
                                        </div>

                                        {/* Convert to MP4 */}
                                        <div className="flex items-center justify-between px-3 py-2 rounded-md border border-border/30 bg-secondary/5">
                                            <div className="space-y-0.5">
                                                <div className="flex items-center gap-2">
                                                    <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Convert to MP4</span>
                                                    <TooltipProvider delayDuration={0}>
                                                        <Tooltip>
                                                            <TooltipTrigger asChild>
                                                                <Info className="w-3 h-3 text-muted-foreground/50 hover:text-foreground cursor-help transition-colors" />
                                                            </TooltipTrigger>
                                                            <TooltipContent className="max-w-[220px] p-2.5 text-xs bg-popover/95 backdrop-blur-sm border-border/50">
                                                                <p className="text-muted-foreground">
                                                                    When disabled, clips are saved as <strong className="text-foreground">.ts</strong> files for faster saving. You can convert them later from the <button onClick={() => window.dispatchEvent(new CustomEvent('open-clips-drawer'))} className="text-primary hover:underline cursor-pointer">clips folder</button>.
                                                                </p>
                                                            </TooltipContent>
                                                        </Tooltip>
                                                    </TooltipProvider>
                                                </div>
                                            </div>
                                            <Switch
                                                checked={config.convertToMP4}
                                                onCheckedChange={(checked) => setConfig(prev => ({ ...prev, convertToMP4: checked }))}
                                                disabled={disabled}
                                                className="scale-90"
                                            />
                                        </div>

                                        {/* FPS & Quality */}
                                        <div className="grid grid-cols-2 gap-4">
                                            <div className="space-y-1.5">
                                                <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider flex items-center gap-1.5">
                                                    <Timer className="w-3 h-3" /> FPS
                                                </label>
                                                <Select
                                                    value={config.fps.toString()}
                                                    onValueChange={(v) => setConfig(prev => ({ ...prev, fps: parseInt(v) }))}
                                                >
                                                    <SelectTrigger className="h-9 bg-accent border-border/50">
                                                        <SelectValue />
                                                    </SelectTrigger>
                                                    <SelectContent>
                                                        {fpsOptions.map(fps => (
                                                            <SelectItem key={fps} value={fps.toString()}>
                                                                {fps}{fps === maxHz ? ' (Native)' : ''}
                                                            </SelectItem>
                                                        ))}
                                                    </SelectContent>
                                                </Select>
                                            </div>
                                            <div className="space-y-1.5">
                                                <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider flex items-center gap-1.5">
                                                    <Sparkles className="w-3 h-3" /> Quality
                                                </label>
                                                <Select
                                                    value={config.bitrate}
                                                    onValueChange={(v) => setConfig(prev => ({ ...prev, bitrate: v }))}
                                                >
                                                    <SelectTrigger className="h-9 bg-accent border-border/50">
                                                        <SelectValue />
                                                    </SelectTrigger>
                                                    <SelectContent>
                                                        <SelectItem value="8M">Medium</SelectItem>
                                                        <SelectItem value="15M">High</SelectItem>
                                                        <SelectItem value="25M">Ultra</SelectItem>
                                                        <SelectItem value="40M">Extreme</SelectItem>
                                                    </SelectContent>
                                                </Select>
                                            </div>
                                        </div>
                                    </div>
                                </ScrollArea>
                            </TabsContent>

                            <TabsContent value="audio" className="animate-in slide-in-from-right-2 duration-300 fade-in-0 mt-0">
                                <ScrollArea className="h-[280px] -mx-4 w-[calc(100%+2rem)]">
                                    <div className="space-y-3 px-4 py-2">
                                        {/* Microphone Selection */}
                                        <div className="space-y-1.5">
                                            <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider flex items-center gap-1.5">
                                                <Mic className="w-3 h-3" /> Microphone
                                            </label>
                                            <div className="flex items-center gap-2">
                                                <Select
                                                    value={config.microphoneDevice || "none"}
                                                    onValueChange={(v) => {
                                                        setConfig(prev => ({ ...prev, microphoneDevice: v === "none" ? "" : v }))
                                                        if (v === "none") setShowMicVolume(false)
                                                    }}
                                                >
                                                    <SelectTrigger className="h-9 bg-accent border-border/50 flex-1">
                                                        <SelectValue placeholder="No Microphone" />
                                                    </SelectTrigger>
                                                    <SelectContent>
                                                        <SelectItem value="none">No Microphone</SelectItem>
                                                        {inputDevices.map(d => (
                                                            <SelectItem key={d} value={d}>
                                                                {d}
                                                            </SelectItem>
                                                        ))}
                                                    </SelectContent>
                                                </Select>

                                                {config.microphoneDevice && config.microphoneDevice !== "none" && (
                                                    <Button
                                                        variant="ghost"
                                                        size="icon"
                                                        className={cn(
                                                            "h-9 w-9 shrink-0 hover:bg-accent hover:text-foreground transition-all duration-200 border border-transparent",
                                                            showMicVolume ? "text-primary bg-accent border-border/50 shadow-sm" : "text-muted-foreground/50"
                                                        )}
                                                        onClick={() => setShowMicVolume(!showMicVolume)}
                                                        title="Adjust Volume"
                                                    >
                                                        <SlidersHorizontal className="w-4 h-4" />
                                                    </Button>
                                                )}
                                            </div>

                                            {/* Mic Volume */}
                                            {showMicVolume && config.microphoneDevice && config.microphoneDevice !== "none" && (
                                                <div className="flex items-center gap-3 px-1 pt-2 pb-1 animate-in fade-in slide-in-from-top-1 duration-200">
                                                    <button
                                                        onClick={() => setConfig(prev => ({ ...prev, micVolume: prev.micVolume === 0 ? 100 : 0 }))}
                                                        className="text-muted-foreground hover:text-foreground transition-colors focus:outline-none"
                                                        title={config.micVolume === 0 ? "Unmute" : "Mute"}
                                                    >
                                                        {config.micVolume === 0 ? <VolumeX className="w-4 h-4" /> :
                                                            config.micVolume < 50 ? <Volume className="w-4 h-4" /> :
                                                                config.micVolume < 100 ? <Volume1 className="w-4 h-4" /> :
                                                                    <Volume2 className="w-4 h-4" />}
                                                    </button>
                                                    <Slider
                                                        value={[config.micVolume ?? 100]}
                                                        min={0}
                                                        max={200}
                                                        step={1}
                                                        onValueChange={([v]) => setConfig(prev => ({ ...prev, micVolume: v }))}
                                                        className="flex-1 cursor-pointer [&_[role=slider]]:h-3.5 [&_[role=slider]]:w-3.5"
                                                    />
                                                    <span className="text-[10px] font-mono w-[3ch] text-right text-muted-foreground select-none">
                                                        {config.micVolume ?? 100}%
                                                    </span>
                                                </div>
                                            )}
                                        </div>

                                        {/* System Audio Selection */}
                                        <div className="space-y-1.5">
                                            <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider flex items-center gap-1.5">
                                                <Settings2 className="w-3 h-3" /> System Audio
                                            </label>
                                            <div className="flex items-center gap-2">
                                                <Select
                                                    value={config.systemAudioDevice || "none"}
                                                    onValueChange={(v) => {
                                                        setConfig(prev => ({ ...prev, systemAudioDevice: v === "none" ? "" : v }))
                                                        if (v === "none") setShowSysVolume(false)
                                                    }}
                                                >
                                                    <SelectTrigger className="h-9 bg-accent border-border/50 flex-1">
                                                        <SelectValue placeholder="No System Audio" />
                                                    </SelectTrigger>
                                                    <SelectContent>
                                                        <SelectItem value="none">No System Audio</SelectItem>
                                                        {outputDevices.map(d => (
                                                            <SelectItem key={d} value={d}>
                                                                {d}
                                                            </SelectItem>
                                                        ))}
                                                    </SelectContent>
                                                </Select>

                                                {config.systemAudioDevice && config.systemAudioDevice !== "none" && (
                                                    <Button
                                                        variant="ghost"
                                                        size="icon"
                                                        className={cn(
                                                            "h-9 w-9 shrink-0 hover:bg-accent hover:text-foreground transition-all duration-200 border border-transparent",
                                                            showSysVolume ? "text-primary bg-accent border-border/50 shadow-sm" : "text-muted-foreground/50"
                                                        )}
                                                        onClick={() => setShowSysVolume(!showSysVolume)}
                                                        title="Adjust Volume"
                                                    >
                                                        <SlidersHorizontal className="w-4 h-4" />
                                                    </Button>
                                                )}
                                            </div>

                                            {/* System Volume */}
                                            {showSysVolume && config.systemAudioDevice && config.systemAudioDevice !== "none" && (
                                                <div className="flex items-center gap-3 px-1 pt-2 pb-1 animate-in fade-in slide-in-from-top-1 duration-200">
                                                    <button
                                                        onClick={() => setConfig(prev => ({ ...prev, sysVolume: prev.sysVolume === 0 ? 100 : 0 }))}
                                                        className="text-muted-foreground hover:text-foreground transition-colors focus:outline-none"
                                                        title={config.sysVolume === 0 ? "Unmute" : "Mute"}
                                                    >
                                                        {config.sysVolume === 0 ? <VolumeX className="w-4 h-4" /> :
                                                            config.sysVolume < 50 ? <Volume className="w-4 h-4" /> :
                                                                config.sysVolume < 100 ? <Volume1 className="w-4 h-4" /> :
                                                                    <Volume2 className="w-4 h-4" />}
                                                    </button>
                                                    <Slider
                                                        value={[config.sysVolume ?? 100]}
                                                        min={0}
                                                        max={200}
                                                        step={1}
                                                        onValueChange={([v]) => setConfig(prev => ({ ...prev, sysVolume: v }))}
                                                        className="flex-1 cursor-pointer [&_[role=slider]]:h-3.5 [&_[role=slider]]:w-3.5"
                                                    />
                                                    <span className="text-[10px] font-mono w-[3ch] text-right text-muted-foreground select-none">
                                                        {config.sysVolume ?? 100}%
                                                    </span>
                                                </div>
                                            )}
                                        </div>
                                    </div>
                                </ScrollArea>
                            </TabsContent>
                        </Tabs>
                    </CardContent>
                </CollapsibleContent>
            </Card>
        </Collapsible >
    )
}
