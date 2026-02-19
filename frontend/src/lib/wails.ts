export interface DisplayInfo {
    index: number
    name: string
    width: number
    height: number
    refreshRate: number
    isPrimary: boolean
}

export interface Config {
    displayIndex: number
    fps: number
    bitrate: number
    recordSeconds: number
    segmentDurationSec: number
    outputDir: string
    microphoneDevice: number
    systemAudioDevice: number
    microphoneVolume: number
    systemAudioVolume: number
    showCursor: boolean
    showBorder: boolean
}

export interface Clip {
    name: string
    path: string
    size: number
    modTime: string
}

export interface State {
    status: 'idle' | 'recording' | 'saving'
    errorMessage?: string
    bufferUsage: number
    recordingFor: number
    diskUsageMB: number
    memoryUsageMB: number
}

import { App as AppBindings } from '../../bindings/rewind/internal'
import { Events } from "@wailsio/runtime"

export const api = {
    async getDisplays(): Promise<DisplayInfo[]> {
        const displays = await AppBindings.ListAvailableDisplays()
        return displays as unknown as DisplayInfo[]
    },

    async getConfig(): Promise<Config> {
        const config = await AppBindings.GetConfig()
        return config as unknown as Config
    },

    async setConfig(config: Config): Promise<void> {
        return AppBindings.UpdateConfig(config as any)
    },

    async getState(): Promise<State> {
        const state = await AppBindings.GetRecordingState()
        return state as unknown as State
    },

    async start(): Promise<void> {
        return AppBindings.StartRecording()
    },

    async stop(): Promise<void> {
        return AppBindings.StopRecording()
    },

    async saveClip(): Promise<string> {
        return AppBindings.SaveCurrentClip()
    },

    async SelectDirectory(): Promise<string> {
        return AppBindings.ChooseOutputDirectory()
    },

    async getClips(): Promise<Clip[]> {
        const clips = await AppBindings.ListSavedClips()
        return clips as unknown as Clip[]
    },

    async openClip(path: string): Promise<void> {
        return AppBindings.OpenClipInExplorer(path)
    },

    async openOutputDirectory(): Promise<void> {
        return AppBindings.OpenOutputDirectory()
    },

    async deleteClips(paths: string[]): Promise<void> {
        return AppBindings.DeleteClips(paths)
    },

    async getInputDevices(): Promise<string[]> {
        return AppBindings.ListAudioInputDevices()
    },

    async getOutputDevices(): Promise<string[]> {
        return AppBindings.ListAudioOutputDevices()
    },

    async refreshConfig(): Promise<void> {
        return AppBindings.RefreshConfig()
    },

    Events: {
        On: (eventName: string, callback: (data: any) => void) => {
            return Events.On(eventName, callback)
        },
        Off: (eventName: string) => {
            return Events.Off(eventName)
        },
        Emit: (eventName: string, data?: any) => {
            return Events.Emit(eventName, data)
        }
    }
}
