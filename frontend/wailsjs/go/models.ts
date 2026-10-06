export namespace main {
	
	export class KillResult {
	    success: boolean;
	    requiresAdmin: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new KillResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.requiresAdmin = source["requiresAdmin"];
	        this.message = source["message"];
	    }
	}
	export class PortInfo {
	    port: number;
	    protocol: string;
	    pid: number;
	    processName: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new PortInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.port = source["port"];
	        this.protocol = source["protocol"];
	        this.pid = source["pid"];
	        this.processName = source["processName"];
	        this.status = source["status"];
	    }
	}

}

