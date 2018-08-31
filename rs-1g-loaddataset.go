// The program tries to exercise read-skew anomaly in google cloud datastore.
//
// Impl:
//     - x = 100, y = 0, invariant x + y = 100
//     - tx A starts and reads x
//     - tx B starts and commits x - 100, y + 100
//     - tx A proceeds and reads y and checks invariant
//
// Results:
//
// Run:
//     $ env DATASTORE_PROJECT_ID=test DATASTORE_NAMESPACE=test ./rs-plain

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
const DatasetSize = int64(1024 * 1024)

type Entity struct {
	Count int    `datastore:"count,noindex"`
	Data  []byte `datastore:"data,noindex"`
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

	fmt.Println("load dataset")
	keys := make([]*datastore.Key, 0, 100)
	entities := make([]*Entity, 0, 100)
	for i := int64(0); i < DatasetSize; i++ {
		keys = keys[:0]
		entities = entities[:0]
		fmt.Println("keys:")
		for j := 0; i < DatasetSize && j < 100; j ++ {
			fmt.Print(" ", i)
			key := datastore.IDKey(Kind, i, nil)
			key.Namespace = Kind
			keys = append(keys, key)
			entities = append(entities, &Entity{Count: 100, Data: make([]byte, 1024)})
			i ++
		}
		fmt.Println(" put multi",len(keys), "keys")
		_, err := clients[0].PutMulti(context.Background(), keys, entities)
		if err != nil {
			panic(err)
		}
	}
}
