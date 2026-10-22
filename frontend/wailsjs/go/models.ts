export namespace main {
	
	export class Category {
	    key: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new Category(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.label = source["label"];
	    }
	}
	export class Entry {
	    id: number;
	    name: string;
	    username: string;
	    password: string;
	    category: string;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.category = source["category"];
	    }
	}
	export class ImportResult {
	    format: string;
	    added: number;
	    skipped: number;
	    invalid: number;
	
	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.format = source["format"];
	        this.added = source["added"];
	        this.skipped = source["skipped"];
	        this.invalid = source["invalid"];
	    }
	}

}

