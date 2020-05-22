package utils

import (
	"fmt"
	"sync"
	"testing"
)

func TestConcurrency(t *testing.T){

	var mutex sync.Mutex
	var group sync.WaitGroup

	peerIdList := []uint32{1,2,3,4,5}
	group.Add(len(peerIdList))

	collectedVote := 1
	voteMap := make(map[uint32]bool)
	voteMap[1] = true

	for _, id := range peerIdList {
		go func(id uint32) {
			mutex.Lock()
			defer func() {
				mutex.Unlock()
				group.Done()
			}()
			if (*&voteMap)[id] == false {
				collectedVote += 1
				(voteMap)[id] = true
			}
		}(id)
	}
	group.Wait()
	fmt.Println(collectedVote)
	fmt.Println(voteMap)
}