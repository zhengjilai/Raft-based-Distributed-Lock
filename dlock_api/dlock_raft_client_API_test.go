package dlock_api

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"
)

// the cli-srv server interface list
// revise here for different environments
// note that we will take turns to request to nodes in addressList

// the local test address list
var addressList = [...]string {
	"0.0.0.0:24005",
	"0.0.0.0:24006",
	"0.0.0.0:24007",
}

// the distributed test address list
//var addressList = [...]string {
//	"121.36.203.158:24005",
//	"121.37.166.51:24005",
//	"121.37.178.20:24005",
//	"121.36.198.5:24005",
//	"121.37.135.56:24005",
//}

func TestDLockRaftClientAPI_PutDelGetState(t *testing.T) {

	fmt.Println("KVTest: Begin to test KVStore Put/Get/Del")

	// sync utils
	var group sync.WaitGroup
	// default request timeout
	timeout := uint32(2500)

	dLockRaftClientAPI := NewDLockRaftClientAPI()
	// You can set logger if you need more details
	dLockRaftClientAPI.Logger = log.New(os.Stdout, "Raft-Dlock-Client-API: ", 0)

	// put state for 20 keys and 100 values
	group.Add(100)
	for i := 0 ; i < 100; i ++ {
		go func(index int) {
			time.Sleep(time.Duration(index * 10) * time.Millisecond)
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

	fmt.Println("DLockTest-NormalRelease: Begin to test DLock normal release")

	var group sync.WaitGroup
	// totally 2 acquirers
	group.Add(2)
	lockName := "DLockReleaseTest"
	indexAccumulate := 0
	startTimestamp := time.Now()

	// dlock client acquirer 1
	go func() {
		dlockClient1 := NewDLockRaftClientAPI()
		dlockClient1.Logger = log.New(os.Stdout, "Raft-Dlock-Client-API: ", 0)
		t.Log(fmt.Printf("DLockTest-NormalRelease: Client 1 has clientId as %s.\n", dlockClient1.ClientId))

		// client 1 acquire dlock at time 0s
		t.Log(fmt.Printf("DLockTest-NormalRelease: Client 1 now at time %d ms.\n", time.Since(startTimestamp).Nanoseconds() / 1000000))
		dlockExpire := int64(6000)
		ok := dlockClient1.AcquireDLock(addressList[indexAccumulate % len(addressList)], lockName, dlockExpire)
		indexAccumulate ++
		if ok {
			t.Log(fmt.Printf("DLockTest-NormalRelease: Client 1 acquire DLock %s succeeded.\n", lockName))
		} else {
			t.Error(fmt.Printf("DLockTest-NormalRelease: Client 1 acquire DLock %s failed.\n", lockName))
			group.Done()
			return
		}

		// wait for 4 seconds
		time.Sleep(4000 * time.Millisecond)

		// client 1 release dlock at time 4s, should succeed
		t.Log(fmt.Printf("DLockTest-NormalRelease: Client 1 now at time %d ms.\n", time.Since(startTimestamp).Nanoseconds() / 1000000))
		dlockInfo1, ok := dlockClient1.QueryDLock(addressList[indexAccumulate % len(addressList)], lockName)
		indexAccumulate ++
		t.Log(fmt.Printf("DLockTest-NormalRelease: Client 1 query DLock %s before release, response %+v.\n",
			lockName, dlockInfo1))
		// client 1 release dlock, should succeed
		ok = dlockClient1.ReleaseDLock(addressList[indexAccumulate % len(addressList)], lockName)
		indexAccumulate ++
		if ok {
			t.Log(fmt.Printf("DLockTest-NormalRelease: Client 1 release DLock %s succeeded.\n", lockName))
		} else {
			t.Error(fmt.Printf("DLockTest-NormalRelease: Client 1 release DLock %s failed.\n", lockName))
			group.Done()
			return
		}
		// test dlock1 state, should have changed
		dlockInfo2, ok := dlockClient1.QueryDLock(addressList[indexAccumulate % len(addressList)], lockName)
		indexAccumulate ++
		if ok && dlockInfo2.Owner == "" {
			t.Log(fmt.Printf("DLockTest-NormalRelease: Client 1 query DLock %s immediately after release, " +
				"the dlock has been released as expected, response %+v.\n",
				lockName, dlockInfo2))
		} else {
			t.Error(fmt.Printf("DLockTest-NormalRelease: Client 1 query DLock %s immediately after release, " +
				"the dlock has not been released as expected, response %+v.\n",
				lockName, dlockInfo2))
			group.Done()
			return
		}

		// client 1 query dlock at time 4.5s
		time.Sleep(500 * time.Millisecond)

		// test dlock1 state, must have changed
		t.Log(fmt.Printf("DLockTest-NormalRelease: Client 1 now at time %d ms.\n", time.Since(startTimestamp).Nanoseconds() / 1000000))
		dlockInfo3, ok := dlockClient1.QueryDLock(addressList[indexAccumulate % len(addressList)], lockName)
		indexAccumulate ++
		if ok && dlockInfo3.Owner == "" {
			t.Log(fmt.Printf("DLockTest-NormalRelease: Client 1 query DLock %s a while after release, " +
				"the dlock has been released as expected, response %+v.\n",
				lockName, dlockInfo3))
		} else {
			t.Error(fmt.Printf("DLockTest-NormalRelease: Client 1 query DLock %s a while after release, " +
				"the dlock has not been released as expected, response %+v.\n",
				lockName, dlockInfo3))
			group.Done()
			return
		}

		group.Done()
	}()

	// dlock client 2
	go func() {
		// start at time 0s
		dlockClient2 := NewDLockRaftClientAPI()
		dlockClient2.Logger = log.New(os.Stdout, "Raft-Dlock-Client-API: ", 0)
		t.Log(fmt.Printf("DLockTest-NormalRelease: Client 2 has clientId as %s.\n", dlockClient2.ClientId))

		// wait for 2 seconds
		time.Sleep(2000 * time.Millisecond)

		// begin to acquire DLock at time 2s
		t.Log(fmt.Printf("DLockTest-NormalRelease: Client 2 now at time %d ms.\n", time.Since(startTimestamp).Nanoseconds() / 1000000))
		dlockExpire := int64(2000)
		// acquire dlock should fail, since it has been acquired by client 1
		// may block about 1 second (as timeout is set as 1000ms)
		ok := dlockClient2.AcquireDLock(addressList[indexAccumulate % len(addressList)],
			lockName, dlockExpire, 1000)
		indexAccumulate ++
		if !ok {
			t.Log(fmt.Printf("DLockTest-NormalRelease: Client 2 acquire DLock %s failed as expected.\n", lockName))
		} else {
			t.Error(fmt.Printf("DLockTest-NormalRelease: Client 2 acquire DLock %s succeeded unexpectedly.\n", lockName))
			group.Done()
			return
		}

		// wait for 4 seconds
		time.Sleep(4000 * time.Millisecond)

		// now Dlock should have been released by client 1, query before acquiring it
		t.Log(fmt.Printf("DLockTest-NormalRelease: Client 2 now at time %d ms.\n", time.Since(startTimestamp).Nanoseconds() / 1000000))
		dlockInfo1, ok := dlockClient2.QueryDLock(addressList[indexAccumulate % len(addressList)], lockName)
		indexAccumulate ++
		if ok && dlockInfo1.Owner == "" {
			t.Log(fmt.Printf("DLockTest-NormalRelease: Client 2 query DLock %s, " +
				"the dlock has been released by Client 1 as expected, response %+v.\n",
				lockName, dlockInfo1))
		} else {
			t.Error(fmt.Printf("DLockTest-NormalRelease: Client 2 query DLock %s, " +
				"the dlock has not been released by Client 1 as expected, response %+v.\n",
				lockName, dlockInfo1))
			group.Done()
			return
		}

		// client 2 try to acquire dlock at about time 7s
		ok = dlockClient2.AcquireDLock(addressList[indexAccumulate % len(addressList)], lockName, dlockExpire)
		indexAccumulate ++
		if ok {
			t.Log(fmt.Printf("DLockTest-NormalRelease: Client 2 acquire DLock %s succeeded.\n", lockName))
		} else {
			t.Error(fmt.Printf("DLockTest-NormalRelease: Client 2 acquire DLock %s failed.\n", lockName))
			group.Done()
			return
		}

		// query dlock immediately after acquiring it, should succeed
		dlockInfo2, ok := dlockClient2.QueryDLock(addressList[indexAccumulate % len(addressList)], lockName)
		indexAccumulate ++
		if dlockInfo2.Owner == dlockClient2.ClientId {
			t.Log(fmt.Printf("DLockTest-NormalRelease: Client 2 query DLock %s after a successful acquirement, " +
				"the dlock has been acquired as expected, response %+v.\n",
				lockName, dlockInfo2))
		} else {
			t.Error(fmt.Printf("DLockTest-NormalRelease: Client 2 query DLock %s after a successful acquirement, " +
				"the dlock has not been acquired as expected, response %+v.\n",
				lockName, dlockInfo2))
			group.Done()
			return
		}

		// release dlock immediately after acquiring it, should succeed
		ok = dlockClient2.ReleaseDLock(addressList[indexAccumulate % len(addressList)], lockName)
		indexAccumulate ++
		if ok {
			t.Log(fmt.Printf("DLockTest-NormalRelease: Client 2 release DLock %s succeeded.\n", lockName))
		} else {
			t.Error(fmt.Printf("DLockTest-NormalRelease: Client 2 release DLock %s failed.\n", lockName))
			group.Done()
			return
		}

		// test dlock state, must have changed for dlock releasing
		t.Log(fmt.Printf("DLockTest-NormalRelease: Client 2 now at time %d ms.\n", time.Since(startTimestamp).Nanoseconds() / 1000000))
		dlockInfo3, ok := dlockClient2.QueryDLock(addressList[indexAccumulate % len(addressList)], lockName)
		indexAccumulate ++
		if ok && dlockInfo3.Owner == "" {
			t.Log(fmt.Printf("DLockTest-NormalRelease: Client 2 query DLock %s after a successful release, " +
				"the dlock has been released as expected, response %+v.\n",
				lockName, dlockInfo3))
		} else {
			t.Error(fmt.Printf("DLockTest-NormalRelease: Client 2 query DLock %s after a successful release, " +
				"the dlock has not been released as expected, response %+v.\n",
				lockName, dlockInfo3))
			group.Done()
			return
		}

		group.Done()
	}()

	group.Wait()
}


func TestDLockExpireAutomatically(t *testing.T)  {

	fmt.Println("DLockTest-AutoExpire: Begin to test DLock expire release")

	var group sync.WaitGroup
	// totally 2 acquirers
	group.Add(2)
	lockName := "DLockExpireTest"
	indexAccumulate := 0
	startTimestamp := time.Now()

	// clients
	dlockClient1 := NewDLockRaftClientAPI()
	dlockClient2 := NewDLockRaftClientAPI()

	// dlock client acquirer 1
	go func() {

		dlockClient1.Logger = log.New(os.Stdout, "Raft-Dlock-Client-API: ", 0)
		t.Log(fmt.Printf("DLockTest-AutoExpire: Client 1 has clientId as %s.\n", dlockClient1.ClientId))

		// client 1 acquire dlock at time 0s
		t.Log(fmt.Printf("DLockTest-AutoExpire: Client 1 now at time %d ms.\n", time.Since(startTimestamp).Nanoseconds() / 1000000))
		dlockExpire := int64(4000)
		ok := dlockClient1.AcquireDLock(addressList[indexAccumulate % len(addressList)], lockName, dlockExpire)
		indexAccumulate ++
		if ok {
			t.Log(fmt.Printf("DLockTest-AutoExpire: Client 1 acquire DLock %s succeeded, expire %d ms\n", lockName, dlockExpire))
		} else {
			t.Error(fmt.Printf("DLockTest-AutoExpire: Client 1 acquire DLock %s failed.\n", lockName))
			group.Done()
			return
		}

		// wait for 8 seconds
		time.Sleep(8000 * time.Millisecond)

		// client 1 query and try to release dlock at time 8s
		t.Log(fmt.Printf("DLockTest-AutoExpire: Client 1 now at time %d ms.\n", time.Since(startTimestamp).Nanoseconds() / 1000000))
		dlockInfo1, ok := dlockClient1.QueryDLock(addressList[indexAccumulate % len(addressList)], lockName)
		indexAccumulate ++
		if ok && dlockInfo1.Owner == dlockClient2.ClientId {
			t.Log(fmt.Printf("DLockTest-AutoExpire: Client 1 query DLock %s, " +
				"the dlock has been released by server automatically and acquired by Client 2 as expected, response %+v.\n",
				lockName, dlockInfo1))
		} else {
			t.Error(fmt.Printf("DLockTest-AutoExpire: Client 1 query DLock %s, " +
				"the dlock has not been released by server automatically and acquired by Client 2 as expected, response %+v.\n",
				lockName, dlockInfo1))
			group.Done()
			return
		}
		// client 1 release dlock at time 8s, should fail
		ok = dlockClient1.ReleaseDLock(addressList[indexAccumulate % len(addressList)], lockName)
		indexAccumulate ++
		if !ok {
			t.Log(fmt.Printf("DLockTest-AutoExpire: Client 1 release DLock %s failed as expected.\n", lockName))
		} else {
			t.Error(fmt.Printf("DLockTest-AutoExpire: Client 1 release DLock %s succeeded unexpectedly.\n", lockName))
			group.Done()
			return
		}

		group.Done()
	}()

	// dlock client 2
	go func() {
		// start at time 0s
		dlockClient2.Logger = log.New(os.Stdout, "Raft-Dlock-Client-API: ", 0)
		t.Log(fmt.Printf("DLockTest-AutoExpire: Client 2 has clientId as %s.\n", dlockClient2.ClientId))

		// wait for 2 seconds
		time.Sleep(2000 * time.Millisecond)

		// begin to acquire DLock at time 2s
		t.Log(fmt.Printf("DLockTest-AutoExpire: Client 2 now at time %d ms.\n", time.Since(startTimestamp).Nanoseconds() / 1000000))
		dlockExpire := int64(4000)
		// acquire dlock should fail, since it has been acquired by client 1
		// may block about 1 second
		ok := dlockClient2.AcquireDLock(addressList[indexAccumulate % len(addressList)],
			lockName, dlockExpire, 1000)
		indexAccumulate ++
		if !ok {
			t.Log(fmt.Printf("DLockTest-AutoExpire: Client 2 acquire DLock %s failed as expected.\n", lockName))
		} else {
			t.Error(fmt.Printf("DLockTest-AutoExpire: Client 2 acquire DLock %s succeeded unexpectedly.\n", lockName))
			group.Done()
			return
		}

		// wait for another 3 seconds
		time.Sleep(3000 * time.Millisecond)

		// now Dlock should have been released automatically by dlock server, query before acquiring it
		t.Log(fmt.Printf("DLockTest-AutoExpire: Client 2 now at time %d ms.\n", time.Since(startTimestamp).Nanoseconds() / 1000000))
		dlockInfo1, ok := dlockClient2.QueryDLock(addressList[indexAccumulate % len(addressList)], lockName)
		indexAccumulate ++
		if ok && dlockInfo1.Owner == "" {
			t.Log(fmt.Printf("DLockTest-AutoExpire: Client 2 query DLock %s, " +
				"the dlock has been released by server automatically as expected, response %+v.\n",
				lockName, dlockInfo1))
		} else {
			t.Error(fmt.Printf("DLockTest-AutoExpire: Client 2 query DLock %s, " +
				"the dlock has not been released by server automatically as expected, response %+v.\n",
				lockName, dlockInfo1))
			group.Done()
			return
		}

		// client 2 try to acquire dlock at about time 6s, should succeed
		ok = dlockClient2.AcquireDLock(addressList[indexAccumulate % len(addressList)], lockName, dlockExpire)
		indexAccumulate ++
		if ok {
			t.Log(fmt.Printf("DLockTest-AutoExpire: Client 2 acquire DLock %s succeeded, expire %d ms.\n", lockName, dlockExpire))
		} else {
			t.Error(fmt.Printf("DLockTest-AutoExpire: Client 2 acquire DLock %s failed.\n", lockName))
			group.Done()
			return
		}

		// wait for another 5 seconds, then dlock should expire
		time.Sleep(5000 * time.Millisecond)

		t.Log(fmt.Printf("DLockTest-AutoExpire: Client 2 now at time %d ms.\n", time.Since(startTimestamp).Nanoseconds() / 1000000))
		dlockInfo3, ok := dlockClient2.QueryDLock(addressList[indexAccumulate % len(addressList)], lockName)
		indexAccumulate ++
		if ok && dlockInfo1.Owner == "" {
			t.Log(fmt.Printf("DLockTest-AutoExpire: Client 2 query DLock %s, " +
				"the dlock has been released by server automatically as expected, response %+v.\n",
				lockName, dlockInfo3))
		} else {
			t.Error(fmt.Printf("DLockTest-AutoExpire: Client 2 query DLock %s, " +
				"the dlock has not been released by server automatically as expected, response %+v.\n",
				lockName, dlockInfo3))
			group.Done()
			return
		}

		t.Log(fmt.Printf("DLockTest-AutoExpire: Client 2 query DLock %s, response %+v.\n",
			lockName, dlockInfo3))

		group.Done()
	}()

	group.Wait()
}

func TestDLockRacing(t *testing.T)  {

	fmt.Println("DLockTest-LockRacing: Begin to test DLock racing")

	var group sync.WaitGroup
	// totally 5 acquirers
	acquirerNum := 10
	group.Add(acquirerNum)
	lockName := "DLockRacingTest"
	indexAccumulate := 0
	startTimestamp := time.Now()
	acquireSequence := 0

	// the first acquirer
	go func() {
		// the client id index
		clientIndex1 := 1
		// the dlock expire for first acquirer, meaning the dlock will expire at time 5s
		dlockExpire1 := int64(5000)

		dlockClient1 := NewDLockRaftClientAPI()
		dlockClient1.Logger = log.New(os.Stdout, "Raft-Dlock-Client-API: ", 0)
		t.Log(fmt.Printf("DLockTest-LockRacing: Client %d has clientId as %s.\n",
			clientIndex1, dlockClient1.ClientId))

		t.Log(fmt.Printf("DLockTest-LockRacing: Client %d now at time %d ms.\n",
			clientIndex1, time.Since(startTimestamp).Nanoseconds() / 1000000))
		ok := dlockClient1.AcquireDLock(addressList[indexAccumulate % len(addressList)], lockName, dlockExpire1)
		indexAccumulate ++
		if ok {
			t.Log(fmt.Printf("DLockTest-LockRacing: Client %d acquire DLock %s succeeded, expire %d ms\n",
				clientIndex1, lockName, dlockExpire1))
		} else {
			t.Error(fmt.Printf("DLockTest-LockRacing: Client %d acquire DLock %s failed.\n", clientIndex1, lockName))
			group.Done()
			return
		}

		// increment sequence
		acquireSequence ++
		if acquireSequence == 1 {
			t.Log(fmt.Printf("DLockTest-LockRacing: Client %d acquire DLock %s as expected, sequence %d.\n",
				clientIndex1, lockName, acquireSequence))
		} else {
			t.Error(fmt.Printf("DLockTest-LockRacing: Client %d acquire DLock %s, sequence %d, " +
				"which is not as expected.\n", clientIndex1, lockName, acquireSequence))
			group.Done()
			return
		}

		group.Done()
	}()


	// the following index acquirer
	dlockExpire := int64(4000)
	for i := 2 ; i < acquirerNum + 1; i++ {

		clientIndexIntermediate := i

		go func() {

			dlockClient := NewDLockRaftClientAPI()
			dlockClient.Logger = log.New(os.Stdout, "Raft-Dlock-Client-API: ", 0)
			t.Log(fmt.Printf("DLockTest-LockRacing: Client %d has clientId as %s.\n",
				clientIndexIntermediate, dlockClient.ClientId))

			// sleep, letting every other node take turns to wake up
			time.Sleep( time.Duration((clientIndexIntermediate - 1) * 300) * time.Millisecond)

			// client i now at time (i-1) * 100 ms, begin to acquire dlock
			t.Log(fmt.Printf("DLockTest-LockRacing: Client %d now at time %d ms.\n",
				clientIndexIntermediate, time.Since(startTimestamp).Nanoseconds() / 1000000))
			ok := dlockClient.AcquireDLock(addressList[indexAccumulate % len(addressList)],
				lockName, dlockExpire, 10000)
			indexAccumulate ++
			if ok {
				t.Log(fmt.Printf("DLockTest-LockRacing: Client %d acquire DLock %s succeeded, expire %d ms\n",
					clientIndexIntermediate, lockName, dlockExpire))
			} else {
				t.Error(fmt.Printf("DLockTest-LockRacing: Client %d acquire DLock %s failed.\n",
					clientIndexIntermediate, lockName))
				group.Done()
				return
			}
			// increment sequence and test it
			// all acquirement will take turns to be processed, which is an implicit function of VolatileAcquirement
			acquireSequence ++
			if acquireSequence == clientIndexIntermediate {
				t.Log(fmt.Printf("DLockTest-LockRacing: Client %d acquire DLock %s, sequence %d.\n",
					clientIndexIntermediate, lockName, acquireSequence))
			} else {
				t.Error(fmt.Printf("DLockTest-LockRacing: Client %d acquire DLock %s, sequence %d, " +
					"which is not as expected.\n", clientIndexIntermediate, lockName, acquireSequence))
				group.Done()
				return
			}
			// release dlock immediately after acquiring it
			ok = dlockClient.ReleaseDLock(addressList[indexAccumulate % len(addressList)], lockName)
			indexAccumulate ++
			if ok {
				t.Log(fmt.Printf("DLockTest-LockRacing: Client %d release DLock %s succeeded as expected.\n",
					clientIndexIntermediate, lockName))
			} else {
				t.Error(fmt.Printf("DLockTest-LockRacing: Client %d release DLock %s failed unexpectedly.\n",
					clientIndexIntermediate, lockName))
				group.Done()
				return
			}

			group.Done()
		}()
	}

	group.Wait()
}