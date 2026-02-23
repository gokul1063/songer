package configs

const DataPath string = "/home/coder/songs/data/"
const LogFormat string = "[%s] - Error : %v\n"
const LogLocation = "/home/coder/songs/logs/"

// {} defines the set of allowed music types
var supportedExt = map[string]struct{}{
	".mp3":  {},
	".wav":  {},
	".flac": {},
}
