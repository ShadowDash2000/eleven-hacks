export namespace elevenlabs {
	
	export class ApiKeyResponse {
	    xi_api_key: string;
	
	    static createFrom(source: any = {}) {
	        return new ApiKeyResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.xi_api_key = source["xi_api_key"];
	    }
	}
	export class DubbingFile {
	    status: string;
	    path: string;
	    name: string;
	    attempt: number;
	    apiKey?: ApiKeyResponse;
	
	    static createFrom(source: any = {}) {
	        return new DubbingFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.path = source["path"];
	        this.name = source["name"];
	        this.attempt = source["attempt"];
	        this.apiKey = this.convertValues(source["apiKey"], ApiKeyResponse);
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

}

