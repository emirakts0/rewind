export namespace app {
	
	export class Config {
	    displayIndex: number;
	    encoderName: string;
	    fps: number;
	    bitrate: string;
	    recordSeconds: number;
	    outputDir: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.displayIndex = source["displayIndex"];
	        this.encoderName = source["encoderName"];
	        this.fps = source["fps"];
	        this.bitrate = source["bitrate"];
	        this.recordSeconds = source["recordSeconds"];
	        this.outputDir = source["outputDir"];
	    }
	}
	export class DisplayInfo {
	    index: number;
	    name: string;
	    width: number;
	    height: number;
	    isPrimary: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DisplayInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.name = source["name"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.isPrimary = source["isPrimary"];
	    }
	}
	export class EncoderInfo {
	    name: string;
	    codec: string;
	    gpuName: string;
	
	    static createFrom(source: any = {}) {
	        return new EncoderInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.codec = source["codec"];
	        this.gpuName = source["gpuName"];
	    }
	}
	export class State {
	    status: string;
	    errorMessage?: string;
	    bufferUsage: number;
	    recordingFor: number;
	
	    static createFrom(source: any = {}) {
	        return new State(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.errorMessage = source["errorMessage"];
	        this.bufferUsage = source["bufferUsage"];
	        this.recordingFor = source["recordingFor"];
	    }
	}

}

