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

}

export namespace main {
	
	export class Token {
	
	
	    static createFrom(source: any = {}) {
	        return new Token(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

