import { Settings2, ChevronUp, Monitor, Timer, Sparkles, Folder, Mic, MousePointer2, Square, Film, Info, Settings, Volume2, RefreshCw, Bell, Video } from 'lucide-react'
import { Switch } from "@/components/ui/switch"
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
import {
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger,
} from "@/components/ui/tooltip"
import { Slider } from "@/components/ui/slider"
import { cn } from '@/lib/utils'
import type { Config, DisplayInfo } from '@/lib/wails'
import { ScrollArea } from "@/components/ui/scroll-area"
import { useState } from 'react'
import { NotificationSettings } from '@/components/notifications'

interface ConfigPanelProps {
    open: boolean
    onOpenChange: (open: boolean) => void
    config: Config
    setConfig: React.Dispatch<React.SetStateAction<Config>>
    displays: DisplayInfo[]
    inputDevices: string[]
    outputDevices: string[]
    disabled?: boolean
    onSelectDirectory: () => void
    onRefreshDevices: () => void
}

const FPS_OPTIONS = [60, 30]

export function ConfigPanel({
    open,
    onOpenChange,
    config,
    setConfig,
    displays,
    inputDevices,
    outputDevices,
    disabled,
    onSelectDirectory,
    onRefreshDevices
}: ConfigPanelProps) {
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
                        <div className="flex items-center gap-2">
                            {open && (
                                <Button
                                    variant="ghost"
                                    size="sm"
                                    className="h-7 w-7 p-0 text-muted-foreground hover:text-foreground"
                                    onClick={(e) => {
                                        e.stopPropagation()
                                        onRefreshDevices()
                                    }}
                                    disabled={disabled}
                                    title="Refresh config & devices"
                                >
                                    <RefreshCw className="w-3.5 h-3.5" />
                                </Button>
                            )}
                            <ChevronUp className={cn(
                                "w-4 h-4 text-muted-foreground/70 transition-transform duration-300",
                                open ? "rotate-180" : "rotate-0"
                            )} />
                        </div>
                    </Button>
                </CollapsibleTrigger>

                <CollapsibleContent>
                    <CardContent className="px-4 pb-0 pt-4 border-t border-border/40">
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
                            <TabsList className="grid w-full grid-cols-3 mb-2">
                                <TabsTrigger value="video" className="gap-1.5">
                                    <Video className="w-3.5 h-3.5" fill="currentColor" />
                                    Video
                                </TabsTrigger>
                                <TabsTrigger value="audio" className="gap-1.5">
                                    <Volume2 className="w-3.5 h-3.5" fill="currentColor" />
                                    Audio
                                </TabsTrigger>
                                <TabsTrigger value="notifications" className="gap-1.5">
                                    <Bell className="w-3.5 h-3.5" fill="currentColor" />
                                    Alerts
                                </TabsTrigger>
                            </TabsList>

                            <TabsContent value="video" className="space-y-4 animate-in slide-in-from-left-2 duration-300 fade-in-0 mt-0">
                                <ScrollArea className="h-[280px] -mx-4 w-[calc(100%+2rem)]">
                                    <div className="space-y-3 px-4 py-2">

                                        <div className="space-y-1.5">
                                            <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider flex items-center gap-1.5">
                                                <Monitor className="w-3 h-3" /> Display
                                            </label>
                                            <Select
                                                value={config.displayIndex.toString()}
                                                onValueChange={(v) => {
                                                    const idx = parseInt(v)
                                                    const display = displays.find(d => d.index === idx)
                                                    setConfig(prev => ({
                                                        ...prev,
                                                        displayIndex: idx,
                                                        monitorName: display?.name || ''
                                                    }))
                                                }}
                                            >
                                                <SelectTrigger className="h-9 bg-accent border-border/50 focus:ring-1 focus:ring-primary/20">
                                                    <SelectValue placeholder="Select display" />
                                                </SelectTrigger>
                                                <SelectContent>
                                                    {displays.map(d => (
                                                        <SelectItem key={d.index} value={d.index.toString()}>
                                                            {d.name}{d.isPrimary ? ' ★' : ''}
                                                        </SelectItem>
                                                    ))}
                                                </SelectContent>
                                            </Select>
                                        </div>

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
                                                        {FPS_OPTIONS.map(fps => (
                                                            <SelectItem key={fps} value={fps.toString()}>
                                                                {fps} FPS
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
                                                    value={config.bitrate.toString()}
                                                    onValueChange={(v) => setConfig(prev => ({ ...prev, bitrate: parseInt(v) }))}
                                                >
                                                    <SelectTrigger className="h-9 bg-accent border-border/50">
                                                        <SelectValue />
                                                    </SelectTrigger>
                                                    <SelectContent>
                                                        <SelectItem value="8">Medium (8 Mbps)</SelectItem>
                                                        <SelectItem value="15">High (15 Mbps)</SelectItem>
                                                        <SelectItem value="25">Ultra (25 Mbps)</SelectItem>
                                                        <SelectItem value="40">Extreme (40 Mbps)</SelectItem>
                                                    </SelectContent>
                                                </Select>
                                            </div>
                                        </div>

                                        <div className="space-y-1.5">
                                            <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider flex items-center gap-1.5">
                                                <Film className="w-3 h-3" /> Segment duration
                                                <TooltipProvider delayDuration={200}>
                                                    <Tooltip>
                                                        <TooltipTrigger asChild>
                                                            <Info className="w-3 h-3 text-muted-foreground/60 hover:text-muted-foreground cursor-help transition-colors" />
                                                        </TooltipTrigger>
                                                        <TooltipContent side="right" className="max-w-[220px] p-2">
                                                            <div className="space-y-1 text-[10px] leading-relaxed normal-case">
                                                                <p className="text-muted-foreground">
                                                                    Duration of each video segment before rotating to the next one.
                                                                </p>
                                                                <p className="text-muted-foreground">
                                                                    <span className="text-green-400">Higher values:</span> Faster saves, less disk fragmentation.
                                                                </p>
                                                                <p className="text-muted-foreground">
                                                                    <span className="text-amber-400">Trade-off:</span> The current recording segment isn't saved, so you may lose up to this duration from the moment you hit save.
                                                                </p>
                                                                <p className="text-primary text-[9px] pt-0.5">5s recommended for balance</p>
                                                            </div>
                                                        </TooltipContent>
                                                    </Tooltip>
                                                </TooltipProvider>
                                            </label>
                                            <Select
                                                value={config.segmentDurationSec.toString()}
                                                onValueChange={(v) => setConfig(prev => ({ ...prev, segmentDurationSec: parseInt(v) }))}
                                            >
                                                <SelectTrigger className="h-9 bg-accent border-border/50">
                                                    <SelectValue />
                                                </SelectTrigger>
                                                <SelectContent>
                                                    <SelectItem value="2">2 seconds</SelectItem>
                                                    <SelectItem value="5">5 seconds (balanced)</SelectItem>
                                                    <SelectItem value="10">10 seconds</SelectItem>
                                                </SelectContent>
                                            </Select>
                                        </div>

                                        <div className="grid grid-cols-2 gap-4">
                                            <div className="flex items-center justify-between px-3 py-2 rounded-md border border-border/30 bg-secondary/5">
                                                <div className="flex items-center gap-2">
                                                    <MousePointer2 className="w-3.5 h-3.5 text-muted-foreground" />
                                                    <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Cursor</span>
                                                </div>
                                                <Switch
                                                    checked={config.showCursor}
                                                    onCheckedChange={(checked) => setConfig(prev => ({ ...prev, showCursor: checked }))}
                                                    disabled={disabled}
                                                    className="scale-90"
                                                />
                                            </div>

                                            <div className="flex items-center justify-between px-3 py-2 rounded-md border border-border/30 bg-secondary/5">
                                                <div className="flex items-center gap-2">
                                                    <Square className="w-3.5 h-3.5 text-muted-foreground" />
                                                    <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Border</span>
                                                </div>
                                                <Switch
                                                    checked={config.showBorder}
                                                    onCheckedChange={(checked) => setConfig(prev => ({ ...prev, showBorder: checked }))}
                                                    disabled={disabled}
                                                    className="scale-90"
                                                />
                                            </div>
                                        </div>
                                    </div>
                                </ScrollArea>
                            </TabsContent>

                            <TabsContent value="audio" className="animate-in slide-in-from-right-2 duration-300 fade-in-0 mt-0">
                                <ScrollArea className="h-[280px] -mx-4 w-[calc(100%+2rem)]">
                                    <div className="space-y-3 px-4 py-2">
                                        <div className="flex items-start gap-2 px-3 py-2 rounded-md bg-muted/30 border border-border/30">
                                            <Info className="w-3.5 h-3.5 text-muted-foreground mt-0.5 flex-shrink-0" />
                                            <p className="text-[10px] text-muted-foreground leading-relaxed">
                                                Volume levels are relative to Windows system settings. Adjust Windows audio mixer for absolute control.
                                            </p>
                                        </div>

                                        <AudioDeviceSelector
                                            label="Microphone"
                                            icon={<Mic className="w-3 h-3" />}
                                            devices={inputDevices}
                                            selectedIndex={config.microphoneDevice}
                                            onSelectDevice={(idx, name) => setConfig(prev => ({
                                                ...prev,
                                                microphoneDevice: idx,
                                                microphoneName: name
                                            }))}
                                            volume={config.microphoneVolume}
                                            onVolumeChange={(vol) => setConfig(prev => ({ ...prev, microphoneVolume: vol }))}
                                            disabled={disabled}
                                        />

                                        <AudioDeviceSelector
                                            label="System audio"
                                            icon={<Settings2 className="w-3 h-3" />}
                                            devices={outputDevices}
                                            selectedIndex={config.systemAudioDevice}
                                            onSelectDevice={(idx, name) => setConfig(prev => ({
                                                ...prev,
                                                systemAudioDevice: idx,
                                                systemAudioName: name
                                            }))}
                                            volume={config.systemAudioVolume}
                                            onVolumeChange={(vol) => setConfig(prev => ({ ...prev, systemAudioVolume: vol }))}
                                            disabled={disabled}
                                        />
                                    </div>
                                </ScrollArea>
                            </TabsContent>

                            <TabsContent value="notifications" className="animate-in slide-in-from-right-2 duration-300 fade-in-0 mt-0">
                                <ScrollArea className="h-[280px] -mx-4 w-[calc(100%+2rem)]">
                                    <div className="px-4 py-2">
                                        <NotificationSettings disabled={disabled} config={config} setConfig={setConfig} />
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

function AudioDeviceSelector({
    label,
    icon,
    devices,
    selectedIndex,
    onSelectDevice,
    volume,
    onVolumeChange,
    disabled
}: {
    label: string
    icon: React.ReactNode
    devices: string[]
    selectedIndex: number
    onSelectDevice: (index: number, deviceName: string) => void
    volume: number
    onVolumeChange: (volume: number) => void
    disabled?: boolean
}) {
    const [showVolume, setShowVolume] = useState(volume !== 100)
    const isDeviceSelected = selectedIndex >= 0

    return (
        <div className="space-y-2">
            <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider flex items-center gap-1.5">
                {icon} {label}
            </label>
            <div className="flex gap-2">
                <Select
                    value={selectedIndex.toString()}
                    onValueChange={(v) => {
                        const idx = parseInt(v)
                        const deviceName = idx >= 0 ? devices[idx] : ''
                        onSelectDevice(idx, deviceName)
                    }}
                    disabled={disabled}
                >
                    <SelectTrigger className="h-9 bg-accent border-border/50 flex-1">
                        <SelectValue placeholder={`No ${label}`} />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectItem value="-1">No {label}</SelectItem>
                        {devices.map((d, idx) => (
                            <SelectItem key={idx} value={idx.toString()}>
                                {d}
                            </SelectItem>
                        ))}
                    </SelectContent>
                </Select>
                {isDeviceSelected && (
                    <Button
                        variant="ghost"
                        size="sm"
                        className="h-9 w-9 p-0 text-muted-foreground hover:text-foreground"
                        onClick={() => setShowVolume(!showVolume)}
                        disabled={disabled}
                    >
                        <Settings className="w-4 h-4" />
                    </Button>
                )}
            </div>
            {isDeviceSelected && showVolume && (
                <div className="pt-2 pb-2">
                    <div className="flex items-center gap-3">
                        <Volume2 className="w-3.5 h-3.5 text-muted-foreground flex-shrink-0" />
                        <Slider
                            value={[volume]}
                            onValueChange={(vals) => onVolumeChange(vals[0])}
                            min={0}
                            max={200}
                            step={5}
                            disabled={disabled}
                            className="flex-1"
                        />
                        <span className="text-xs font-mono text-muted-foreground w-10 text-right">{volume}%</span>
                    </div>
                </div>
            )}
        </div>
    )
}
