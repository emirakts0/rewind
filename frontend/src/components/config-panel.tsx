import { Settings2, ChevronUp, Monitor, Timer, Sparkles, Folder, Mic, MousePointer2, Square } from 'lucide-react'
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
import { cn } from '@/lib/utils'
import type { Config, DisplayInfo } from '@/lib/wails'
import { ScrollArea } from "@/components/ui/scroll-area"
import { useMemo } from 'react'

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
}

// Standard FPS options
const STANDARD_FPS = [60, 30, 24]

export function ConfigPanel({
    open,
    onOpenChange,
    config,
    setConfig,
    displays,
    inputDevices,
    outputDevices,
    disabled,
    onSelectDirectory
}: ConfigPanelProps) {
    // Get current display's refresh rate
    const selectedDisplay = displays.find(d => d.index === config.displayIndex)
    const maxHz = selectedDisplay?.refreshRate || 60

    // Build FPS options: native Hz + standard options below it
    const fpsOptions = useMemo(() => {
        const options = new Set<number>()
        options.add(maxHz)
        STANDARD_FPS.forEach(fps => {
            if (fps <= maxHz) options.add(fps)
        })
        return Array.from(options).sort((a, b) => b - a)
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

                                        {/* Cursor & Border */}
                                        <div className="space-y-2">
                                            <div className="flex items-center justify-between px-3 py-2 rounded-md border border-border/30 bg-secondary/5">
                                                <div className="flex items-center gap-2">
                                                    <MousePointer2 className="w-3.5 h-3.5 text-muted-foreground" />
                                                    <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Show Cursor</span>
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
                                                    <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Show Border</span>
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
                                        {/* Microphone Selection */}
                                        <AudioDeviceSelector
                                            label="Microphone"
                                            icon={<Mic className="w-3 h-3" />}
                                            devices={inputDevices}
                                            selectedIndex={config.microphoneDevice}
                                            onSelectDevice={(idx) => setConfig(prev => ({ ...prev, microphoneDevice: idx }))}
                                            disabled={disabled}
                                        />

                                        {/* System Audio Selection */}
                                        <AudioDeviceSelector
                                            label="System Audio"
                                            icon={<Settings2 className="w-3 h-3" />}
                                            devices={outputDevices}
                                            selectedIndex={config.systemAudioDevice}
                                            onSelectDevice={(idx) => setConfig(prev => ({ ...prev, systemAudioDevice: idx }))}
                                            disabled={disabled}
                                        />
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

// Audio Device Selector Component
function AudioDeviceSelector({
    label,
    icon,
    devices,
    selectedIndex,
    onSelectDevice,
    disabled
}: {
    label: string
    icon: React.ReactNode
    devices: string[]
    selectedIndex: number
    onSelectDevice: (index: number) => void
    disabled?: boolean
}) {
    return (
        <div className="space-y-1.5">
            <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider flex items-center gap-1.5">
                {icon} {label}
            </label>
            <Select
                value={selectedIndex.toString()}
                onValueChange={(v) => onSelectDevice(parseInt(v))}
                disabled={disabled}
            >
                <SelectTrigger className="h-9 bg-accent border-border/50">
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
        </div>
    )
}
