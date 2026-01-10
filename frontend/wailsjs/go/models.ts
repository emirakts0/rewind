export namespace app {
	
	export class Clip {
	    name: string;
	    path: string;
	    size: number;
	    // Go type: time
	    modTime: any;
	
	    static createFrom(source: any = {}) {
	        return new Clip(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.size = source["size"];
	        this.modTime = this.convertValues(source["modTime"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Config {
	    displayIndex: number;
	    encoderName: string;
	    fps: number;
	    bitrate: string;
	    recordSeconds: number;
	    outputDir: string;
	    convertToMP4: boolean;
	
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
	        this.convertToMP4 = source["convertToMP4"];
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

