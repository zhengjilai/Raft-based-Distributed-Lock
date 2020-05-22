package utils

func Uint64Min(a uint64, b uint64) uint64 {
	if a < b {
		return a
	} else {
		return b
	}
}

func Uint64Max(a uint64, b uint64) uint64 {
	if a < b {
		return b
	} else {
		return a
	}
}

func NumberInUint32List(list []uint32, num uint32) bool {
	result := false
	for _ , item := range list {
		if num == item {
			result = true
		}
	}
	return result
}