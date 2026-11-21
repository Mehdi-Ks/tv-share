export namespace main {
	
	export class DepsStatus {
	    rclone: boolean;
	    ffmpeg: boolean;
	    winget: boolean;
	    ready: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new DepsStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rclone = source["rclone"];
	        this.ffmpeg = source["ffmpeg"];
	        this.winget = source["winget"];
	        this.ready = source["ready"];
	        this.message = source["message"];
	    }
	}
	export class SubOption {
	    path: string;
	    label: string;
	    lang: string;
	    sdh: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SubOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.label = source["label"];
	        this.lang = source["lang"];
	        this.sdh = source["sdh"];
	    }
	}
	export class Movie {
	    path: string;
	    title: string;
	    sizeMB: number;
	    hasSubs: boolean;
	    subs: SubOption[];
	    defaultSub: number;
	    outside: boolean;
	    added: number;
	
	    static createFrom(source: any = {}) {
	        return new Movie(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.title = source["title"];
	        this.sizeMB = source["sizeMB"];
	        this.hasSubs = source["hasSubs"];
	        this.subs = this.convertValues(source["subs"], SubOption);
	        this.defaultSub = source["defaultSub"];
	        this.outside = source["outside"];
	        this.added = source["added"];
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
	export class MovieInfo {
	    minutes: number;
	    resolution: string;
	    video: string;
	    audio: string[];
	    subtitles: string[];
	
	    static createFrom(source: any = {}) {
	        return new MovieInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.minutes = source["minutes"];
	        this.resolution = source["resolution"];
	        this.video = source["video"];
	        this.audio = source["audio"];
	        this.subtitles = source["subtitles"];
	    }
	}
	export class MovieMeta {
	    id: number;
	    title: string;
	    year: string;
	    overview: string;
	    genres: string[];
	    rating: number;
	    votes: number;
	    url: string;
	    posterPath: string;
	    backdropPath: string;
	    fetched: number;
	    manual: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MovieMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.year = source["year"];
	        this.overview = source["overview"];
	        this.genres = source["genres"];
	        this.rating = source["rating"];
	        this.votes = source["votes"];
	        this.url = source["url"];
	        this.posterPath = source["posterPath"];
	        this.backdropPath = source["backdropPath"];
	        this.fetched = source["fetched"];
	        this.manual = source["manual"];
	    }
	}
	export class NetworkInfo {
	    name: string;
	    category: string;
	    ip: string;
	
	    static createFrom(source: any = {}) {
	        return new NetworkInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.category = source["category"];
	        this.ip = source["ip"];
	    }
	}
	export class PrepareResult {
	    output: string;
	    recycled: boolean;
	    syncNote: string;
	
	    static createFrom(source: any = {}) {
	        return new PrepareResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.output = source["output"];
	        this.recycled = source["recycled"];
	        this.syncNote = source["syncNote"];
	    }
	}
	export class Settings {
	    folder: string;
	    name: string;
	    port: number;
	    autoStartShare: boolean;
	    launchAtLogin: boolean;
	    recycleOriginal: boolean;
	    subLang: string;
	    setupDone: boolean;
	    tmdb: string;
	    tmdbKey: string;
	    noAutoSync: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.folder = source["folder"];
	        this.name = source["name"];
	        this.port = source["port"];
	        this.autoStartShare = source["autoStartShare"];
	        this.launchAtLogin = source["launchAtLogin"];
	        this.recycleOriginal = source["recycleOriginal"];
	        this.subLang = source["subLang"];
	        this.setupDone = source["setupDone"];
	        this.tmdb = source["tmdb"];
	        this.tmdbKey = source["tmdbKey"];
	        this.noAutoSync = source["noAutoSync"];
	    }
	}
	export class TMDBState {
	    available: boolean;
	    builtIn: boolean;
	    ownKey: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TMDBState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.builtIn = source["builtIn"];
	        this.ownKey = source["ownKey"];
	    }
	}
	export class State {
	    version: string;
	    deps: DepsStatus;
	    sharing: boolean;
	    settings: Settings;
	    network: NetworkInfo;
	    busy: boolean;
	    tmdb: TMDBState;
	
	    static createFrom(source: any = {}) {
	        return new State(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.deps = this.convertValues(source["deps"], DepsStatus);
	        this.sharing = source["sharing"];
	        this.settings = this.convertValues(source["settings"], Settings);
	        this.network = this.convertValues(source["network"], NetworkInfo);
	        this.busy = source["busy"];
	        this.tmdb = this.convertValues(source["tmdb"], TMDBState);
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
	
	export class TMDBCandidate {
	    id: number;
	    title: string;
	    year: string;
	    poster: string;
	
	    static createFrom(source: any = {}) {
	        return new TMDBCandidate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.year = source["year"];
	        this.poster = source["poster"];
	    }
	}

}

