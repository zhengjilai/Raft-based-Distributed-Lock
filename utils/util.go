package utils

import (
	"math/rand"
	"time"
)

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

func IndexInUint32List(list []uint32, num uint32) int {
	if !NumberInUint32List(list, num) {
		return -1
	} else {
		result := 0
		for i , item := range list {
			if num == item {
				result = i
			}
		}
		return result
	}
}

func RandomObjectInStringList(list []string) string {
	if len(list) == 0 {
		return ""
	} else {
		rand.Seed(time.Now().Unix())
		randIndex := rand.Intn(len(list))
		return list[randIndex]
	}
}