// The program tries to exercise reads blocking writes behavior.
// Plus checks for repeatable reads in transactions.
//
// Impl:
//     - A: read X
//     - B: write X in sep tx and commit
//     - A: commit
//
// Results:
//     - repeatable reads present
//     - reads does not block writes
//
// Run:
//     $ env DATASTORE_PROJECT_ID=test DATASTORE_NAMESPACE=test ./r-block-w

package main

import (
	"context"
	"fmt"
	"os"

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

	// set initial state
	_, err := clients[0].RunInTransaction(context.Background(), func(tx *datastore.Transaction) error {
		if err := write(tx, "x", &Entity{Count: 0}); err != nil {
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

	step := 0
	for {
		step++
		fmt.Println("step = ", step)
		// proceed
		_, err = clients[0].RunInTransaction(context.Background(), txA)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read tx failed: %s\n", err.Error())
		}
	}
}

func txA(tx *datastore.Transaction) error {
	fmt.Println("A: start")
	defer fmt.Println("A: done")

	x0, err := read(tx, "x")
	if err != nil {
		return fmt.Errorf("A: read x0: %s", err.Error())
	}

	_, err = clients[1].RunInTransaction(context.Background(), txB)
	if err != nil {
		fmt.Fprintf(os.Stderr, "A: executing B: %s\n", err.Error())
	}

	x1, err := read(tx, "x")
	if err != nil {
		return fmt.Errorf("A: read x1: %s", err.Error())
	}

	fmt.Println("A: read x0=", x0.Count, "x1=", x1.Count)

	// check invariant
	if x0.Count != x1.Count {
		panic("A: no repeatable reads")
	}

	return nil
}

func txB(tx *datastore.Transaction) error {
	fmt.Println("B: start")
	defer fmt.Println("B: done")

	x, err := read(tx, "x")
	if err != nil {
		return fmt.Errorf("B: read X: %s", err.Error())
	}
	if err = write(tx, "x", &Entity{Count: x.Count + 1}); err != nil {
		return fmt.Errorf("B: write X: %s", err.Error())
	}

	fmt.Println("B: write x=", x.Count+1)

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
