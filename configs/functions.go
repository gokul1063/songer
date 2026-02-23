package configs

func IsSupported(ext string) bool {
	_, exist := supportedExt[ext]
	return exist
}
