// The program tries to exercise read-skew anomaly in google cloud datastore.
//
// Impl:
//     - x = 100, y = 0, invariant x + y = 100
//     - tx A starts and reads x
//     - tx B starts and commits x - 100, y + 100
//     - tx A proceeds and reads y and checks invariant
//
// Results:
//     - rpc error: code = Aborted desc = too much contention on these datastore entities. please try again. entity group key: (app=e~test!test, test_read_skew, "x")
//
// Run:
//     $ env DATASTORE_PROJECT_ID=test DATASTORE_NAMESPACE=test ./rs-plain

package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"cloud.google.com/go/datastore"
)

var ProjectID = os.Getenv("DATASTORE_PROJECT_ID")
var Namespace = os.Getenv("DATASTORE_NAMESPACE")

const Kind = "test_read_skew"

type Entity struct {
	Count int `datastore:"count,noindex"`
}

var clients = make([]*datastore.Client, 2)

func main() {
	for i := 0; i < len(clients); i++ {
		client, err := datastore.NewClient(context.Background(), ProjectID)
		if err != nil {
			panic(err)
		}
		clients[i] = client
	}

	step := 0
	for {
		step++
		fmt.Println("step = ", step)

		// set initial state
		_, err := clients[0].RunInTransaction(context.Background(), func(tx *datastore.Transaction) error {
			if err := write(tx, "x", &Entity{Count: 100}); err != nil {
				return err
			}
			if err := write(tx, "y", &Entity{Count: 0}); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			panic(err)
		}

		// proceed
		_, err = clients[0].RunInTransaction(context.Background(), txA)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read tx failed: %s\n", err.Error())
		}
	}
}

func txA(tx *datastore.Transaction) error {
	x, err := read(tx, "x")
	if err != nil {
		return fmt.Errorf("read tx: error reading x: %s", err.Error())
	}

	time.Sleep(3 * time.Second)

	var ws sync.WaitGroup
	ws.Add(1)
	go func() {
		_, err = clients[1].RunInTransaction(context.Background(), txB)
		if err != nil {
			fmt.Fprintf(os.Stderr, "write tx failed: %s\n", err.Error())
		}
		ws.Done()
	}()
	ws.Wait()

	time.Sleep(3 * time.Second)

	y, err := read(tx, "y")
	if err != nil {
		return fmt.Errorf("read tx: error reading y: %s", err.Error())
	}

	fmt.Println("read tx read x=", x.Count, "y=", y.Count)
	// check invariant
	if x.Count+y.Count != 100 {
		panic("read skew")
	}

	return nil
}

func txB(tx *datastore.Transaction) error {
	x, err := read(tx, "x")
	if err != nil {
		return err
	}
	y, err := read(tx, "y")
	if err != nil {
		return err
	}

	fmt.Println("write tx read x=", x.Count, "y=", y.Count)
	if x.Count+y.Count != 100 {
		panic("invalid invariant")
	}

	if err = write(tx, "x", &Entity{Count: x.Count - 100}); err != nil {
		return nil
	}
	if err = write(tx, "y", &Entity{Count: y.Count + 100}); err != nil {
		return nil
	}

	return nil
}

func write(tx *datastore.Transaction, key string, e *Entity) error {
	k := datastore.NameKey(Kind, key, nil)
	k.Namespace = Namespace
	_, err := tx.Put(k, e)
	return err
}

func read(tx *datastore.Transaction, key string) (*Entity, error) {
	var e = &Entity{}
	k := datastore.NameKey(Kind, key, nil)
	k.Namespace = Namespace
	return e, tx.Get(k, e)
}
