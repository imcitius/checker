package status

func IsLoud() bool {
	if MainStatus != "quiet" {
		return true
	}
	return false
}
