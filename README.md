# datastore (ds) write skew (ws)

Research of various behaviors of the Google Cloud Datastore database transactions isolations.

# scenario

> In a write skew anomaly, two transactions (T1 and T2) concurrently read an overlapping data set (e.g. values V1 and V2), concurrently make disjoint updates (e.g. T1 updates V1, T2 updates V2), and finally concurrently commit, neither having seen the update performed by the other.

[Snapshot isolation, Wiki.](https://en.wikipedia.org/wiki/Snapshot_isolation)

# tests

## ws-plain.go

Small dataset of 2 keys.

Reading 2 keys and making disjoint writes to them issuing an increment mutation.

Results: does not demonstrate write-skew anomaly. all transactions that are
to be committed after one disjoint writes has been made first are aborted and
retried reading result of the first transaction.

Log:

```
    step = 1
    t= 4 id1= 0 id2= 0 id2++
    t= 2 id1= 0 id2= 0 id2++
    t= 3 id1= 0 id2= 0 id2++
    t= 5 id1= 0 id2= 0 id1++
    t= 0 id1= 0 id2= 0 id2++
    t= 6 id1= 0 id2= 0 id2++
    t= 8 id1= 0 id2= 0 id1++
    t= 1 id1= 0 id2= 0 id2++
    t= 7 id1= 0 id2= 0 id1++
    t= 9 id1= 0 id2= 0 id1++
    t= 8 id1= 0 id2= 1
    t= 3 id1= 0 id2= 1
    t= 7 id1= 0 id2= 1
    t= 1 id1= 0 id2= 1
    t= 0 id1= 0 id2= 1
    t= 5 id1= 0 id2= 1
    t= 9 id1= 0 id2= 1
    t= 2 id1= 0 id2= 1
    t= 6 id1= 0 id2= 1
    id1= 0 id2= 1
    step = 2
    ...
```

## rs-plain.go

Small dataset of 2 keys.

Transaction A does a prefix read checking for an invariant,
another transaction B commits a change in the middle of the first one.

Results: does not demonstrate read-skew anomaly due to the:
`rpc error: code = Aborted desc = too much contention on these datastore entities. please try again. entity group key: (app=e~test!test, test_read_skew, "x")`.
Datastore detects read-skew when reading second variable Y in reading
transaction B right after the writing transaction A(X, Y) commits.

Log:

```
    step =  1
    write tx read x= 100 y= 0
    read tx failed: read tx: error reading y: rpc error: code = Aborted desc = too much contention on these datastore entities. please try again. entity group key: (app=e~test!test, test_read_skew, "x")
    step =  2
    write tx read x= 100 y= 0
    read tx failed: read tx: error reading y: rpc error: code = Aborted desc = too much contention on these datastore entities. please try again. entity group key: (app=e~test!test, test_read_skew, "x")
    step =  3
    ...
```

## rs-1g.go

Dataset of 1M of entities 1024B in size.

Transaction A does a prefix read checking for an invariant, another transaction B
commits a change in the middle of the first one.

Results: does not demonstrate read-skew anomaly due to the:
`rpc error: code = Aborted desc = too much contention on these datastore entities. please try again. entity group key: (app=e~test!test, test_read_skew, 1)`.
Datastore detects read-skew when reading second variable Y in reading
transaction B right after the writing transaction A(X, Y) commits.

## r-block-w.go

The program tries to exercise reads blocking writes behavior and
checks repeatable reads isolation.

Transaction A reads a variable, after what the B writes to that variable
and then both commits.

Results: Repeatable reads are present. Reads does not block writes.

Log:

```
    step =  1
    A: start
    B: start
    B: write x= 1
    B: done
    A: read x0= 0 x1= 0
    A: done
    step =  2
    A: start
    B: start
    B: write x= 2
    B: done
    A: read x0= 1 x1= 1
    A: done
    ...
```

## Results

The tests for detecting write-skew and read-skew anomalies did not
demonstrate expected behavior. Tests failed to produce those anomalies.

From the documentation it is known that datastore transactions are
internally serializable and externally are closer to the read committed
isolation.

The datastore behavior is rather similar to the 2 phase locking (2PL)
in order to implement proper transaction isolation, but as far as
the serializable isolation is not guaranteed it's worth of writing
read/write blocking tests.
