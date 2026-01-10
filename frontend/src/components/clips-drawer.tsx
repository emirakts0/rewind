import { useEffect, useState } from 'react'
import { motion, AnimatePresence } from "framer-motion"
import { FolderOpen, FileVideo, Clock, RefreshCcw, FileDigit, ArrowLeft, MoreHorizontal, Film, Loader2 } from 'lucide-react'
import { api, type Clip } from '@/lib/wails'
import { cn } from '@/lib/utils'

import {
    Sheet,
    SheetContent,
    SheetDescription,
    SheetHeader,
    SheetTitle,
    SheetTrigger,
} from "@/components/ui/sheet"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Button } from "@/components/ui/button"
import { toast } from "sonner"
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

export function ClipsDrawer() {
    const [clips, setClips] = useState<Clip[]>([])
    const [loading, setLoading] = useState(false)
    const [open, setOpen] = useState(false)
    const [converting, setConverting] = useState<Record<string, boolean>>({})

    const fetchClips = async () => {
        setLoading(true)
        try {
            const data = await api.getClips()
            // Sort by modTime desc (newest first)
            const sorted = (data || []).sort((a, b) =>
                new Date(b.modTime).getTime() - new Date(a.modTime).getTime()
            )
            setClips(sorted)
        } catch (err) {
            console.error(err)
            toast.error("Failed to load clips")
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        if (open) {
            fetchClips()
        }
    }, [open])

    const handleOpenClip = async (path: string) => {
        try {
            await api.openClip(path)
        } catch (err) {
            toast.error("Failed to open clip")
        }
    }

    const formatSize = (bytes: number) => {
        if (bytes === 0) return '0 B'
        const k = 1024
        const sizes = ['B', 'KB', 'MB', 'GB']
        const i = Math.floor(Math.log(bytes) / Math.log(k))
        return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
    }

    const formatDate = (dateStr: string) => {
        return new Date(dateStr).toLocaleString()
    }

    const getIcon = (name: string) => {
        if (name.endsWith('.ts')) return <FileDigit className="h-5 w-5 text-amber-500/70" />
        return <FileVideo className="h-5 w-5 text-primary" />
    }

    const handleConvert = async (path: string) => {
        setConverting(prev => ({ ...prev, [path]: true }))
        try {
            await api.convertToMP4(path)
            toast.success("Converted successfully")
            fetchClips()
        } catch (err) {
            toast.error("Conversion failed")
            console.error(err)
        } finally {
            setConverting(prev => ({ ...prev, [path]: false }))
        }
    }

    return (
        <Sheet open={open} onOpenChange={setOpen}>
            <SheetTrigger asChild>
                <Button
                    variant="ghost"
                    size="icon"
                    className="h-9 w-9 text-muted-foreground hover:text-foreground rounded-full hover:bg-white/10"
                >
                    <FolderOpen className="h-5 w-5" />
                </Button>
            </SheetTrigger>
            <SheetContent className="w-screen max-w-none h-full border-l-0 [&>button]:hidden pt-5 px-6">
                <SheetHeader className="mb-4">
                    <SheetTitle className="flex items-center gap-4 text-xl font-black tracking-tighter">
                        <Button
                            variant="ghost"
                            size="icon"
                            className="h-8 w-8 text-muted-foreground hover:text-foreground -ml-2"
                            onClick={() => setOpen(false)}
                        >
                            <ArrowLeft className="h-5 w-5" />
                        </Button>
                        <span>Library</span>
                        <Button
                            variant="ghost"
                            size="icon"
                            className="h-6 w-6 text-muted-foreground hover:text-foreground ml-auto"
                            onClick={fetchClips}
                            disabled={loading}
                        >
                            <RefreshCcw className={cn("h-3.5 w-3.5", loading && "animate-spin")} />
                        </Button>
                    </SheetTitle>
                    <SheetDescription className="hidden">
                        All your captured moments.
                    </SheetDescription>
                </SheetHeader>

                <ScrollArea className="h-[calc(100vh-80px)] -mx-6">
                    {clips.length === 0 ? (
                        <div className="flex flex-col items-center justify-center py-20 text-muted-foreground">
                            <FileVideo className="h-12 w-12 opacity-10 mb-4" />
                            <p className="text-sm font-medium">No clips found</p>
                        </div>
                    ) : (
                        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2 px-4 pb-10">
                            <AnimatePresence mode="popLayout" initial={false}>
                                {clips.map((clip) => (
                                    <motion.div
                                        key={clip.path}
                                        layout
                                        initial={{ opacity: 0, scale: 0.9 }}
                                        animate={{ opacity: 1, scale: 1 }}
                                        exit={{ opacity: 0, scale: 0.9 }}
                                        transition={{
                                            opacity: { duration: 0.2 },
                                            layout: { duration: 0.3, type: "spring", bounce: 0.2 }
                                        }}
                                        className="group relative flex items-center gap-3 p-3 rounded-lg border border-border/40 bg-card/50 hover:bg-accent/50"
                                    >
                                        <button
                                            onClick={() => handleOpenClip(clip.path)}
                                            className="flex-1 flex items-center gap-3 min-w-0 text-left"
                                        >
                                            <div className="h-10 w-10 rounded-md bg-secondary/50 flex items-center justify-center shrink-0">
                                                {getIcon(clip.name)}
                                            </div>
                                            <div className="flex-1 min-w-0">
                                                <p className="font-medium text-sm truncate">{clip.name}</p>
                                                <div className="flex items-center gap-2 text-[10px] text-muted-foreground/80 mt-0.5">
                                                    <span className="flex items-center gap-1 bg-background/50 px-1.5 py-0.5 rounded">
                                                        <Clock className="w-2.5 h-2.5" />
                                                        {formatDate(clip.modTime)}
                                                    </span>
                                                    <span className="font-mono">{formatSize(clip.size)}</span>
                                                </div>
                                            </div>
                                        </button>

                                        {/* Actions for TS files */}
                                        {clip.name.endsWith('.ts') && (
                                            <DropdownMenu>
                                                <DropdownMenuTrigger asChild>
                                                    <Button
                                                        variant="ghost"
                                                        size="icon"
                                                        className="h-8 w-8 opacity-0 group-hover:opacity-100 transition-opacity focus:opacity-100 data-[state=open]:opacity-100"
                                                        disabled={converting[clip.path]}
                                                    >
                                                        {converting[clip.path] ? (
                                                            <Loader2 className="h-4 w-4 animate-spin" />
                                                        ) : (
                                                            <MoreHorizontal className="h-4 w-4" />
                                                        )}
                                                    </Button>
                                                </DropdownMenuTrigger>
                                                <DropdownMenuContent align="end">
                                                    <DropdownMenuItem onClick={() => handleConvert(clip.path)}>
                                                        <Film className="w-4 h-4 mr-2" />
                                                        Convert to MP4
                                                    </DropdownMenuItem>
                                                </DropdownMenuContent>
                                            </DropdownMenu>
                                        )}
                                    </motion.div>
                                ))}
                            </AnimatePresence>
                        </div>
                    )}
                </ScrollArea>
            </SheetContent>
        </Sheet>
    )
}
