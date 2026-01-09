import { Settings2, ChevronUp, Monitor, Cpu, Timer, Sparkles, Folder, Mic, Volume2 } from 'lucide-react'
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
import type { Config, DisplayInfo, EncoderInfo } from '@/lib/wails'
import { Slider } from "@/components/ui/slider"

interface ConfigPanelProps {
    open: boolean
    onOpenChange: (open: boolean) => void
    config: Config
    setConfig: React.Dispatch<React.SetStateAction<Config>>
    displays: DisplayInfo[]
    encoders: EncoderInfo[]
    disabled?: boolean
    onSelectDirectory: () => void
}

export function ConfigPanel({
    open,
    onOpenChange,
    config,
    setConfig,
    displays,
    encoders,
    disabled,
    onSelectDirectory
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
                        <ChevronUp className={cn(
                            "w-4 h-4 text-muted-foreground/70 transition-transform duration-300",
                            open ? "rotate-180" : "rotate-0"
                        )} />
                    </Button>
                </CollapsibleTrigger>

                <CollapsibleContent>
                    <CardContent className="px-4 pb-4 pt-2 border-t border-border/40">
                        <Tabs defaultValue="video" className="w-full">
                            <TabsList className="grid w-full grid-cols-2 mb-4">
                                <TabsTrigger value="video">Video</TabsTrigger>
                                <TabsTrigger value="audio">Audio</TabsTrigger>
                            </TabsList>

                            <TabsContent value="video" className="space-y-4 animate-in slide-in-from-left-2 duration-300 fade-in-0">
                                {/* Output Directory */}
                                <div className="space-y-2">
                                    <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider flex items-center gap-1.5">
                                        <Folder className="w-3 h-3" /> Output Folder
                                    </label>
                                    <Button
                                        variant="outline"
                                        size="sm"
                                        onClick={onSelectDirectory}
                                        className="w-full justify-start text-left font-normal bg-secondary/30 border-border/50 h-9 truncate px-3 text-muted-foreground hover:text-foreground hover:bg-secondary/50"
                                        title={config.outputDir}
                                    >
                                        {config.outputDir || "./clips"}
                                    </Button>
                                </div>

                                {/* Display Select */}
                                <div className="space-y-2">
                                    <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider flex items-center gap-1.5">
                                        <Monitor className="w-3 h-3" /> Display
                                    </label>
                                    <Select
                                        value={config.displayIndex.toString()}
                                        onValueChange={(v) => setConfig(prev => ({ ...prev, displayIndex: parseInt(v) }))}
                                    >
                                        <SelectTrigger className="h-9 bg-secondary/30 border-border/50 focus:ring-1 focus:ring-primary/20">
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
                                <div className="space-y-2">
                                    <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider flex items-center gap-1.5">
                                        <Cpu className="w-3 h-3" /> Encoder
                                    </label>
                                    <Select
                                        value={config.encoderName}
                                        onValueChange={(v) => setConfig(prev => ({ ...prev, encoderName: v }))}
                                    >
                                        <SelectTrigger className="h-9 bg-secondary/30 border-border/50">
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

                                {/* FPS & Quality */}
                                <div className="grid grid-cols-2 gap-4">
                                    <div className="space-y-2">
                                        <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider flex items-center gap-1.5">
                                            <Timer className="w-3 h-3" /> FPS
                                        </label>
                                        <Select
                                            value={config.fps.toString()}
                                            onValueChange={(v) => setConfig(prev => ({ ...prev, fps: parseInt(v) }))}
                                        >
                                            <SelectTrigger className="h-9 bg-secondary/30 border-border/50">
                                                <SelectValue />
                                            </SelectTrigger>
                                            <SelectContent>
                                                <SelectItem value="30">30</SelectItem>
                                                <SelectItem value="60">60</SelectItem>
                                                <SelectItem value="120">120</SelectItem>
                                            </SelectContent>
                                        </Select>
                                    </div>
                                    <div className="space-y-2">
                                        <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider flex items-center gap-1.5">
                                            <Sparkles className="w-3 h-3" /> Quality
                                        </label>
                                        <Select
                                            value={config.bitrate}
                                            onValueChange={(v) => setConfig(prev => ({ ...prev, bitrate: v }))}
                                        >
                                            <SelectTrigger className="h-9 bg-secondary/30 border-border/50">
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
                            </TabsContent>

                            <TabsContent value="audio" className="space-y-4 animate-in slide-in-from-right-2 duration-300 fade-in-0">
                                <div className="p-4 border border-dashed border-border/50 rounded-lg bg-secondary/10">
                                    <div className="flex flex-col items-center justify-center text-center space-y-3 py-4">
                                        <div className="w-10 h-10 rounded-full bg-secondary/20 flex items-center justify-center">
                                            <Mic className="w-5 h-5 text-muted-foreground" />
                                        </div>
                                        <div className="space-y-1">
                                            <h4 className="text-sm font-medium">Audio Capture</h4>
                                            <p className="text-xs text-muted-foreground max-w-[200px]">
                                                Audio settings will be available in a future update.
                                            </p>
                                        </div>
                                    </div>
                                </div>

                                {/* Placeholder for UI visualization */}
                                <div className="space-y-4 opacity-50 pointer-events-none filter blur-[1px]">
                                    <div className="space-y-2">
                                        <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider flex items-center gap-1.5">
                                            <Mic className="w-3 h-3" /> Input Source
                                        </label>
                                        <Select disabled>
                                            <SelectTrigger className="h-9 bg-secondary/30 border-border/50">
                                                <SelectValue placeholder="Default Microphone" />
                                            </SelectTrigger>
                                        </Select>
                                    </div>
                                    <div className="space-y-2">
                                        <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider flex items-center gap-1.5">
                                            <Volume2 className="w-3 h-3" /> System Volume
                                        </label>
                                        <div className="flex items-center gap-3">
                                            <Slider defaultValue={[80]} max={100} step={1} className="flex-1" />
                                            <span className="text-xs font-medium tabular-nums w-8">80%</span>
                                        </div>
                                    </div>
                                </div>
                            </TabsContent>
                        </Tabs>
                    </CardContent>
                </CollapsibleContent>
            </Card>
        </Collapsible>
    )
}
