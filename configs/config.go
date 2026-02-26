package configs

const DataPath string = "/home/coder/songs/data/"
const LogFormat string = "[%s] - Error : %v\n"
const LogLocation string = "/home/coder/songs/logs/"
const SocketPath string = "/tmp/mpv.sock"

// {} defines the set of allowed music types
var supportedExt = map[string]struct{}{
	".mp3":  {},
	".wav":  {},
	".flac": {},
}
