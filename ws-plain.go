// The program shows a behavior of a simple, classic write-skew anomaly example
// in which two transactions read some data and then perform disjoint writes.
//
// Impl:
//     - x = y = 0
//     - read x, y; if x + y < 1 then z++, where z = {x, y}[tx_id]
//     - read x, y; check x + y < 2
//
// Results:
//     - database detects that the second transaction can't be committed due
//       something and restarts it.
//
// Run:
//     $ env DATASTORE_PROJECT_ID=test DATASTORE_NAMESPACE=test ./ws-plain

package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sync"

	"cloud.google.com/go/datastore"
)

var ProjectID = os.Getenv("DATASTORE_PROJECT_ID")
var Namespace = os.Getenv("DATASTORE_NAMESPACE")

const Kind = "test_write_skew"
const Parallelism = 10

type Entity struct {
	Count int `datastore:"count,noindex"`
}

func main() {
	var clients []*datastore.Client
	for i := 0; i < Parallelism; i++ {
		client, err := datastore.NewClient(context.Background(), ProjectID)
		if err != nil {
			panic(err)
		}
		clients = append(clients, client)
	}

	step := 0
	for {
		step++
		fmt.Println("step = ", step)

		// set initial state
		_, err := clients[0].RunInTransaction(context.Background(), func(tx *datastore.Transaction) error {
			if err := write(tx, "id1", &Entity{Count: 0}); err != nil {
				return err
			}
			if err := write(tx, "id2", &Entity{Count: 0}); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			panic(err)
		}

		// try to cause write-skew
		var w sync.WaitGroup
		w.Add(Parallelism)
		for i := 0; i < Parallelism; i++ {
			go func(id int, t int) {
				_, err = clients[id].RunInTransaction(context.Background(), func(tx *datastore.Transaction) error {
					id1, err := read(tx, "id1")
					if err != nil {
						return err
					}
					id2, err := read(tx, "id2")
					if err != nil {
						return err
					}
					if id1.Count+id2.Count < 1 {
						switch id {
						case 0:
							fmt.Println("t=", t, "id1=", id1.Count, "id2=", id2.Count, "id1++")
							if err = write(tx, "id1", &Entity{Count: id1.Count + 1}); err != nil {
								fmt.Fprintf(os.Stderr, "%d put failed: %s\n", i, err.Error())
								return err
							}
						case 1:
							fmt.Println("t=", t, "id1=", id1.Count, "id2=", id2.Count, "id2++")
							if err = write(tx, "id2", &Entity{Count: id2.Count + 1}); err != nil {
								fmt.Fprintf(os.Stderr, "%d put failed: %s\n", i, err.Error())
								return err
							}
						default:
							panic("invalid index")
						}
					} else {
						fmt.Println("t=", t, "id1=", id1.Count, "id2=", id2.Count)
					}
					return nil
				})
				if err != nil {
					fmt.Fprintf(os.Stderr, "%d tx failed: %s\n", i, err.Error())
				}

				w.Done()
			}(rand.Intn(2), i)
		}
		w.Wait()

		// check
		_, err = clients[0].RunInTransaction(context.Background(), func(tx *datastore.Transaction) error {
			id1, err := read(tx, "id1")
			if err != nil {
				return err
			}
			id2, err := read(tx, "id2")
			if err != nil {
				return err
			}
			if id1.Count+id2.Count >= 2 {
				panic("write skew found")
			}
			fmt.Println("id1=", id1.Count, "id2=", id2.Count)
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "tx failed: %s\n", err.Error())
		}
	}
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
