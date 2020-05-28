package client

import (
	"strconv"
	"testing"
	"time"
)

func TestDLockRaftClientAPI_PutDelGetState(t *testing.T) {

	dLockRaftClientAPI := NewDLockRaftClientAPI()
	address := "0.0.0.0:24006"
	timeout := uint32(2500)

	// put state for 20 keys and 100 values
	for i := 0 ; i < 100; i ++ {
		time.Sleep(3 * time.Millisecond)
		dLockRaftClientAPI.PutState(address, strconv.Itoa(i % 20), []byte(strconv.Itoa(i)), timeout)
	}
	// delete 10 keys, 20-24 are invalid keys
	for i := 15 ; i < 25; i ++ {
		time.Sleep(3 * time.Millisecond)
		dLockRaftClientAPI.DelState(address, strconv.Itoa(i), timeout)
	}
	// get state for 15 keys, 15-24 has been deleted
	for i := 10; i < 25; i ++ {
		time.Sleep(3 * time.Millisecond)
		dLockRaftClientAPI.DelState(address, strconv.Itoa(i), timeout)
	}


}
