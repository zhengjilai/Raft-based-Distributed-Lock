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
	dLockRaftClientAPI := NewDLockRaftClientAPI()
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
				fmt.Printf("KVTest: PutState, Key %d succeeded.\n", index)
			} else {
				fmt.Printf("KVTest: PutState, Key %d failed.\n", index)
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
				fmt.Printf("KVTest: DelState, Key %d succeeded.\n", index)
			} else {
				fmt.Printf("KVTest: DelState, Key %d failed.\n", index)
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
				fmt.Printf("KVTest: GetState, Key: %d, Value %d.\n", index, digit)
			} else {
				fmt.Printf("KVTest: GetState, Key: %d has no availble value.\n", index)
			}
			group.Done()
		}(i)
	}
	group.Wait()
}

func TestDLockRaftClientAPI_AcquireQueryReleaseDLock(t *testing.T) {

	var group sync.WaitGroup
	addressList := [...]string {
		"0.0.0.0:24005",
		"0.0.0.0:24006",
		"0.0.0.0:24007",
	}
	// totally 3 acquirers
	group.Add(3)

	// dlock acquirer 1
	go func() {
		dlockClient1 := NewDLockRaftClientAPI()
		dlockExpire := int64(8000)
		dlockClient1.AcquireDLock(addressList[0], "dlock1", dlockExpire)
		dlockClient1.QueryDLock(addressList[1], "dlock1")

		time.Sleep(4000 * time.Millisecond)

		dlockClient1.QueryDLock(addressList[2], "dlock1")
		dlockClient1.QueryDLock(addressList[1], "dlock2")
		dlockClient1.ReleaseDLock(addressList[2], "dlock1")

		group.Done()
	}()

	// dlock acquirer 2
	go func() {
		dlockClient2 := NewDLockRaftClientAPI()
		dlockExpire := int64(2000)
		dlockClient2.AcquireDLock(addressList[1], "dlock1", dlockExpire)

		time.Sleep(4000 * time.Millisecond)

		dlockClient2.QueryDLock(addressList[0], "dlock1")
		dlockClient2.QueryDLock(addressList[1], "dlock2")
		dlockClient2.ReleaseDLock(addressList[2], "dlock1")

		group.Done()
	}()

	// dlock acquirer 3
	go func() {
		dlockClient3 := NewDLockRaftClientAPI()
		dlockExpire := int64(4000)
		dlockClient3.AcquireDLock(addressList[2], "dlock1", dlockExpire)

		time.Sleep(4000 * time.Millisecond)

		dlockClient3.QueryDLock(addressList[2], "dlock1")
		dlockClient3.QueryDLock(addressList[1], "dlock2")
		dlockClient3.ReleaseDLock(addressList[2], "dlock1")

		group.Done()
	}()

	group.Wait()
}
