// Wails bindings types
export interface DisplayInfo {
    index: number
    name: string
    width: number
    height: number
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
}

export interface State {
    status: 'idle' | 'recording' | 'saving' | 'error'
    errorMessage?: string
    bufferUsage: number
    recordingFor: number
}

// Wails runtime bindings
declare global {
    interface Window {
        go: {
            app: {
                App: {
                    Initialize(): Promise<void>
                    GetDisplays(): Promise<DisplayInfo[]>
                    GetEncoders(): Promise<EncoderInfo[]>
                    GetConfig(): Promise<Config>
                    SetConfig(config: Config): Promise<void>
                    GetState(): Promise<State>
                    Start(): Promise<void>
                    Stop(): Promise<void>
                    SaveClip(): Promise<string>
                    IsRecording(): Promise<boolean>
                }
            }
        }
    }
}

// API wrapper with error handling
export const api = {
    async initialize(): Promise<void> {
        return window.go.app.App.Initialize()
    },

    async getDisplays(): Promise<DisplayInfo[]> {
        return window.go.app.App.GetDisplays()
    },

    async getEncoders(): Promise<EncoderInfo[]> {
        return window.go.app.App.GetEncoders()
    },

    async getConfig(): Promise<Config> {
        return window.go.app.App.GetConfig()
    },

    async setConfig(config: Config): Promise<void> {
        return window.go.app.App.SetConfig(config)
    },

    async getState(): Promise<State> {
        return window.go.app.App.GetState()
    },

    async start(): Promise<void> {
        return window.go.app.App.Start()
    },

    async stop(): Promise<void> {
        return window.go.app.App.Stop()
    },

    async saveClip(): Promise<string> {
        return window.go.app.App.SaveClip()
    },

    async isRecording(): Promise<boolean> {
        return window.go.app.App.IsRecording()
    },
}
