export namespace client {
	
	export class AgentDisposition {
	    pacing: string;
	    depth: string;
	    tone: string;
	    initiative: string;
	
	    static createFrom(source: any = {}) {
	        return new AgentDisposition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pacing = source["pacing"];
	        this.depth = source["depth"];
	        this.tone = source["tone"];
	        this.initiative = source["initiative"];
	    }
	}
	export class Agent {
	    agent_id: string;
	    status: string;
	    disposition?: AgentDisposition;
	    channels?: string[];
	
	    static createFrom(source: any = {}) {
	        return new Agent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agent_id = source["agent_id"];
	        this.status = source["status"];
	        this.disposition = this.convertValues(source["disposition"], AgentDisposition);
	        this.channels = source["channels"];
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
	
	export class Model {
	    id: string;
	    provider: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new Model(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.provider = source["provider"];
	        this.name = source["name"];
	    }
	}
	export class PortDefinition {
	    name: string;
	    type: string;
	    required?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PortDefinition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.required = source["required"];
	    }
	}
	export class ProviderInfo {
	    name: string;
	    version: string;
	    capabilities?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ProviderInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.capabilities = source["capabilities"];
	    }
	}
	export class Provider {
	    name: string;
	    status: string;
	    info?: ProviderInfo;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new Provider(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.status = source["status"];
	        this.info = this.convertValues(source["info"], ProviderInfo);
	        this.error = source["error"];
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
	
	export class Skill {
	    id: string;
	    name: string;
	    description?: string;
	    version?: string;
	    category?: string;
	    plugin?: string;
	    inputs: PortDefinition[];
	    outputs: PortDefinition[];
	    cas_hash?: string;
	
	    static createFrom(source: any = {}) {
	        return new Skill(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.version = source["version"];
	        this.category = source["category"];
	        this.plugin = source["plugin"];
	        this.inputs = this.convertValues(source["inputs"], PortDefinition);
	        this.outputs = this.convertValues(source["outputs"], PortDefinition);
	        this.cas_hash = source["cas_hash"];
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

export namespace main {
	
	export class AppConfig {
	    gateway_url: string;
	    version: string;
	    session_id: string;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.gateway_url = source["gateway_url"];
	        this.version = source["version"];
	        this.session_id = source["session_id"];
	    }
	}

}

