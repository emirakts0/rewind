export interface DisplayInfo {
    index: number
    name: string
    width: number
    height: number
    refreshRate: number
    isPrimary: boolean
}

export interface EncoderInfo {
    name: string
    codec: string
    gpuName: string
}

export interface Config {
    displayIndex: number
    encoderName: string
    fps: number
    bitrate: string
    recordSeconds: number
    outputDir: string
    convertToMP4: boolean
    microphoneDevice: string
    micVolume: number
    systemAudioDevice: string
    sysVolume: number
}

export interface Clip {
    name: string
    path: string
    size: number
    modTime: string
    isRawFolder: boolean
    durationSec?: number
}

export interface State {
    status: 'idle' | 'recording' | 'saving' | 'error'
    errorMessage?: string
    bufferUsage: number
    recordingFor: number
}

import * as AppBindings from '../../bindings/rewind/internal/app/app'
import { Events } from "@wailsio/runtime"

export const api = {
    async initialize(): Promise<void> {
        return AppBindings.Initialize()
    },

    async getDisplays(): Promise<DisplayInfo[]> {
        const displays = await AppBindings.GetDisplays()
        return displays as unknown as DisplayInfo[]
    },

    async getEncoders(): Promise<EncoderInfo[]> {
        const encoders = await AppBindings.GetEncodersForDisplay(0)
        return encoders as unknown as EncoderInfo[]
    },

    async getConfig(): Promise<Config> {
        const config = await AppBindings.GetConfig()
        return config as unknown as Config
    },

    async setConfig(config: Config): Promise<void> {
        return AppBindings.SetConfig(config as any)
    },

    async getState(): Promise<State> {
        const state = await AppBindings.GetState()
        return state as unknown as State
    },

    async start(): Promise<void> {
        return AppBindings.Start()
    },

    async stop(): Promise<void> {
        return AppBindings.Stop()
    },

    async saveClip(): Promise<string> {
        return AppBindings.SaveClip()
    },

    async isRecording(): Promise<boolean> {
        return AppBindings.IsRecording()
    },

    async SelectDirectory(): Promise<string> {
        return AppBindings.SelectDirectory()
    },

    async estimateMemory(bitrate: string, seconds: number, hasMic: boolean, hasSys: boolean): Promise<string> {
        return AppBindings.EstimateMemory(bitrate, seconds, hasMic, hasSys)
    },

    async getClips(): Promise<Clip[]> {
        const clips = await AppBindings.GetClips()
        return clips as unknown as Clip[]
    },

    async openClip(path: string): Promise<void> {
        return AppBindings.OpenClip(path)
    },

    async convertToMP4(path: string): Promise<void> {
        return AppBindings.ConvertToMP4(path)
    },

    async getEncodersForDisplay(displayIndex: number): Promise<EncoderInfo[]> {
        const encoders = await AppBindings.GetEncodersForDisplay(displayIndex)
        return encoders as unknown as EncoderInfo[]
    },

    async getInputDevices(): Promise<string[]> {
        return (AppBindings as any).GetInputDevices()
    },

    async getOutputDevices(): Promise<string[]> {
        return (AppBindings as any).GetOutputDevices()
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
