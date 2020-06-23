package client

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"
)

// the cli-srv server interface list
// revise here for different environments
// note that we will take turns to request to nodes in addressList
var addressList = [...]string {
	"0.0.0.0:24005",
	"0.0.0.0:24006",
	"0.0.0.0:24007",
}

func TestDLockRaftClientAPI_PutDelGetState(t *testing.T) {

	var group sync.WaitGroup

	timeout := uint32(2500)
	dLockRaftClientAPI := NewDLockRaftClientAPI()

	// put state for 20 keys and 100 values
	group.Add(100)
	for i := 0 ; i < 100; i ++ {
		go func(index int) {
			time.Sleep(time.Duration(index * 5) * time.Millisecond)
			success := dLockRaftClientAPI.PutState(addressList[index % len(addressList)],
				strconv.Itoa(index % 20), []byte(strconv.Itoa(index)))
			if success {
				t.Log(fmt.Printf("KVTest: PutState, Key %d succeeded.\n", index))
			} else {
				t.Error(fmt.Printf("KVTest: PutState, Key %d failed.\n", index))
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
			success := dLockRaftClientAPI.DelState(addressList[index % len(addressList)], strconv.Itoa(index))
			if success && index <= 19{
				t.Log(fmt.Printf("KVTest: DelState, Key %d succeeded.\n", index))
			} else {
				t.Log(fmt.Printf("KVTest: DelState, Key %d failed.\n", index))
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
			value := dLockRaftClientAPI.GetState(addressList[index % len(addressList)], strconv.Itoa(index), timeout)
			if value != nil {
				digit, _ := strconv.Atoi(string(value))
				t.Log(fmt.Printf("KVTest: GetState, Key: %d, Value %d.\n", index, digit))
			} else {
				t.Log(fmt.Printf("KVTest: GetState, Key: %d has no availble value.\n", index))
			}
			group.Done()
		}(i)
	}
	group.Wait()
}


func TestNormalDLockRelease(t *testing.T)  {

	fmt.Println("Begin to test DLock normal release")

	var group sync.WaitGroup
	// totally 2 acquirers
	group.Add(2)
	lockName := "DLockReleaseTest"
	indexAccumulate := 0

	// dlock client acquirer 1
	go func() {
		dlockClient1 := NewDLockRaftClientAPI()

		// client 1 acquire dlock at time 0s
		dlockExpire := int64(6000)
		ok := dlockClient1.AcquireDLock(addressList[indexAccumulate % len(addressList)], lockName, dlockExpire)
		indexAccumulate ++
		if ok {
			t.Log(fmt.Printf("DLockTest: Client 1 acquire DLock %s succeeded.\n", lockName))
		} else {
			t.Error(fmt.Printf("DLockTest: Client 1 acquire DLock %s failed.\n", lockName))
			group.Done()
			return
		}

		// wait for 4 seconds
		time.Sleep(4000 * time.Millisecond)

		// client 1 release dlock at time 4s, should succeed
		dlockInfo1, ok := dlockClient1.QueryDLock(addressList[indexAccumulate % len(addressList)], lockName)
		indexAccumulate ++
		t.Log(fmt.Printf("DLockTest: Client 1 query DLock %s before release, response %+v.\n",
			lockName, dlockInfo1))
		// client 1 release dlock, should succeed
		ok = dlockClient1.ReleaseDLock(addressList[indexAccumulate % len(addressList)], lockName)
		indexAccumulate ++
		if ok {
			t.Log(fmt.Printf("DLockTest: Client 1 release DLock %s succeeded.\n", lockName))
		} else {
			t.Error(fmt.Printf("DLockTest: Client 1 release DLock %s failed.\n", lockName))
			group.Done()
			return
		}
		// test dlock1 state, may have not changed for network latency
		dlockInfo2, ok := dlockClient1.QueryDLock(addressList[indexAccumulate % len(addressList)], lockName)
		indexAccumulate ++
		t.Log(fmt.Printf("DLockTest: Client 1 query DLock %s immediately after release, response %+v.\n",
			lockName, dlockInfo2))

		// client 1 query dlock at time 4.5s
		time.Sleep(500 * time.Millisecond)
		// test dlock1 state, must have changed for network latency
		dlockInfo3, ok := dlockClient1.QueryDLock(addressList[indexAccumulate % len(addressList)], lockName)
		indexAccumulate ++
		t.Log(fmt.Printf("DLockTest: Client 1 query DLock %s a while after release, response %+v.\n",
			lockName, dlockInfo3))

		group.Done()
	}()

	// dlock client 2
	go func() {
		// start at time 0s
		dlockClient2 := NewDLockRaftClientAPI()

		// wait for 2 seconds
		time.Sleep(2000 * time.Millisecond)

		// begin to acquire DLock at time 2s
		dlockExpire := int64(2000)
		// acquire dlock should fail, since it has been acquired by client 1
		// may block about 1 second
		ok := dlockClient2.AcquireDLock(addressList[indexAccumulate % len(addressList)],
			lockName, dlockExpire, 1000)
		indexAccumulate ++
		if !ok {
			t.Log(fmt.Printf("DLockTest: Client 2 acquire DLock %s failed as expected.\n", lockName))
		} else {
			t.Error(fmt.Printf("DLockTest: Client 2 acquire DLock %s succeeded unexpectedly.\n", lockName))
			group.Done()
			return
		}

		// wait for 4 seconds
		time.Sleep(4000 * time.Millisecond)

		// now Dlock should have been released by client 1, query before acquiring it
		dlockInfo1, ok := dlockClient2.QueryDLock(addressList[indexAccumulate % len(addressList)], lockName)
		indexAccumulate ++
		t.Log(fmt.Printf("DLockTest: Client 2 query DLock %s, response %+v.\n",
			lockName, dlockInfo1))
		// client 2 try to acquire dlock at about time 7s
		ok = dlockClient2.AcquireDLock(addressList[indexAccumulate % len(addressList)], lockName, dlockExpire)
		indexAccumulate ++
		if ok {
			t.Log(fmt.Printf("DLockTest: Client 2 acquire DLock %s succeeded.\n", lockName))
		} else {
			t.Error(fmt.Printf("DLockTest: Client 2 acquire DLock %s failed.\n", lockName))
			group.Done()
			return
		}
		// release dlock immediately after acquiring it, should succeed
		ok = dlockClient2.ReleaseDLock(addressList[indexAccumulate % len(addressList)], lockName)
		indexAccumulate ++
		if ok {
			t.Log(fmt.Printf("DLockTest: Client 2 release DLock %s succeeded.\n", lockName))
		} else {
			t.Error(fmt.Printf("DLockTest: Client 2 release DLock %s failed.\n", lockName))
		}
		group.Done()
	}()

	group.Wait()
}


func TestNormalDLockExpire(t *testing.T)  {

	fmt.Println("Begin to test DLock expire release")

	var group sync.WaitGroup
	// totally 2 acquirers
	group.Add(2)
	lockName := "DLockExpireTest"
	indexAccumulate := 0
	startTimestamp := time.Now()

	// dlock client acquirer 1
	go func() {
		dlockClient1 := NewDLockRaftClientAPI()
		t.Log(fmt.Printf("DLockTest: Client 1 has clientId as %s.\n", dlockClient1.ClientId))

		// client 1 acquire dlock at time 0s
		t.Log(fmt.Printf("DLockTest: Client 1 now at time %d ms.\n", time.Since(startTimestamp).Nanoseconds() / 1000000))
		dlockExpire := int64(4000)
		ok := dlockClient1.AcquireDLock(addressList[indexAccumulate % len(addressList)], lockName, dlockExpire)
		indexAccumulate ++
		if ok {
			t.Log(fmt.Printf("DLockTest: Client 1 acquire DLock %s succeeded, expire %d ms\n", lockName, dlockExpire))
		} else {
			t.Error(fmt.Printf("DLockTest: Client 1 acquire DLock %s failed.\n", lockName))
			group.Done()
			return
		}

		// wait for 8 seconds
		time.Sleep(8000 * time.Millisecond)

		// client 1 release dlock at time 8s
		t.Log(fmt.Printf("DLockTest: Client 1 now at time %d ms.\n", time.Since(startTimestamp).Nanoseconds() / 1000000))
		dlockInfo1, ok := dlockClient1.QueryDLock(addressList[indexAccumulate % len(addressList)], lockName)
		indexAccumulate ++
		t.Log(fmt.Printf("DLockTest: Client 1 query DLock %s before release, response %+v.\n",
			lockName, dlockInfo1))
		// client 1 release dlock at time 8s, should fail
		ok = dlockClient1.ReleaseDLock(addressList[indexAccumulate % len(addressList)], lockName)
		indexAccumulate ++
		if !ok {
			t.Log(fmt.Printf("DLockTest: Client 1 release DLock %s failed as expected.\n", lockName))
		} else {
			t.Error(fmt.Printf("DLockTest: Client 1 release DLock %s succeeded unexpectedly.\n", lockName))
			group.Done()
			return
		}

		// client 1 query dlock at time 8.5s
		time.Sleep(500 * time.Millisecond)

		// test dlock1 state, must have changed for network latency
		t.Log(fmt.Printf("DLockTest: Client 1 now at time %d ms.\n", time.Since(startTimestamp).Nanoseconds() / 1000000))
		dlockInfo2, ok := dlockClient1.QueryDLock(addressList[indexAccumulate % len(addressList)], lockName)
		indexAccumulate ++
		t.Log(fmt.Printf("DLockTest: Client 1 query DLock %s a while after release, response %+v.\n",
			lockName, dlockInfo2))

		group.Done()
	}()

	// dlock client 2
	go func() {
		// start at time 0s
		dlockClient2 := NewDLockRaftClientAPI()
		t.Log(fmt.Printf("DLockTest: Client 2 has clientId as %s.\n", dlockClient2.ClientId))

		// wait for 2 seconds
		time.Sleep(2000 * time.Millisecond)

		// begin to acquire DLock at time 2s
		t.Log(fmt.Printf("DLockTest: Client 2 now at time %d ms.\n", time.Since(startTimestamp).Nanoseconds() / 1000000))
		dlockExpire := int64(4000)
		// acquire dlock should fail, since it has been acquired by client 1
		// may block about 1 second
		ok := dlockClient2.AcquireDLock(addressList[indexAccumulate % len(addressList)],
			lockName, dlockExpire, 1000)
		indexAccumulate ++
		if !ok {
			t.Log(fmt.Printf("DLockTest: Client 2 acquire DLock %s failed as expected.\n", lockName))
		} else {
			t.Error(fmt.Printf("DLockTest: Client 2 acquire DLock %s succeeded unexpectedly.\n", lockName))
			group.Done()
			return
		}

		// wait for another 3 seconds
		time.Sleep(3000 * time.Millisecond)

		// now Dlock should have been released by client 1, query before acquiring it
		t.Log(fmt.Printf("DLockTest: Client 2 now at time %d ms.\n", time.Since(startTimestamp).Nanoseconds() / 1000000))
		dlockInfo1, ok := dlockClient2.QueryDLock(addressList[indexAccumulate % len(addressList)], lockName)
		indexAccumulate ++
		t.Log(fmt.Printf("DLockTest: Client 2 query DLock %s, response %+v.\n",
			lockName, dlockInfo1))
		// client 2 try to acquire dlock at about time 6s, should succeed
		ok = dlockClient2.AcquireDLock(addressList[indexAccumulate % len(addressList)], lockName, dlockExpire)
		indexAccumulate ++
		if ok {
			t.Log(fmt.Printf("DLockTest: Client 2 acquire DLock %s succeeded, expire %d ms.\n", lockName, dlockExpire))
		} else {
			t.Error(fmt.Printf("DLockTest: Client 2 acquire DLock %s failed.\n", lockName))
			group.Done()
			return
		}

		// wait for another 5 seconds, then dlock should expire
		time.Sleep(5000 * time.Millisecond)

		t.Log(fmt.Printf("DLockTest: Client 2 now at time %d ms.\n", time.Since(startTimestamp).Nanoseconds() / 1000000))
		dlockInfo3, ok := dlockClient2.QueryDLock(addressList[indexAccumulate % len(addressList)], lockName)
		indexAccumulate ++
		t.Log(fmt.Printf("DLockTest: Client 2 query DLock %s, response %+v.\n",
			lockName, dlockInfo3))

		group.Done()
	}()

	group.Wait()
}