package client

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestDLockRaftClientAPI_PutDelGetState(t *testing.T) {

	var group sync.WaitGroup

	timeout := uint32(2500)
	clientId := "Tester"
	dLockRaftClientAPI := NewDLockRaftClientAPI(clientId)
	addressList := [...]string {
		"0.0.0.0:24005",
		"0.0.0.0:24006",
		"0.0.0.0:24007",
	}

	// put state for 20 keys and 100 values
	group.Add(100)
	for i := 0 ; i < 100; i ++ {
		go func(index int) {
			time.Sleep(time.Duration(index * 5) * time.Millisecond)
			success := dLockRaftClientAPI.PutState(addressList[index % 3],
				strconv.Itoa(index % 20), []byte(strconv.Itoa(index)))
			if success {
				fmt.Printf("GetState, Key %d succeeded", index)
			} else {
				fmt.Printf("GetState, Key %d failed", index)
			}
			group.Done()
		}(i)
	}
	group.Wait()

	// delete 10 keys, 20-24 are invalid keys
	group.Add(10)
	for i := 15 ; i < 25; i ++ {
		go func(index int) {
			time.Sleep(time.Duration(rand.Intn(300)) * time.Millisecond)
			success := dLockRaftClientAPI.DelState(addressList[index % 3], strconv.Itoa(index))
			if success {
				fmt.Printf("DelState, Key %d succeeded", index)
			} else {
				fmt.Printf("DelState, Key %d failed", index)
			}
			group.Done()
		}(i)
	}
	group.Wait()

	// get state for 15 keys, 15-24 has been deleted
	group.Add(15)
	for i := 10; i < 25; i ++ {
		go func(index int) {
			time.Sleep(time.Duration(rand.Intn(300)) * time.Millisecond)
			value := dLockRaftClientAPI.GetState(addressList[index % 3], strconv.Itoa(index), timeout)
			if value != nil {
				digit, _ := strconv.Atoi(string(value))
				fmt.Printf("GetState, Key: %d, Value %d", index, digit)
			} else {
				fmt.Printf("GetState, Key: %d has no availble value.", index)
			}
			group.Done()
		}(i)
	}
	group.Wait()
}
