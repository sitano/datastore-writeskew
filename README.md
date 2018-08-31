# datastore (ds) write skew (ws)

Research of various behaviors of the Google Cloud Datastore database transactions isolations.

# scenario

> In a write skew anomaly, two transactions (T1 and T2) concurrently read an overlapping data set (e.g. values V1 and V2), concurrently make disjoint updates (e.g. T1 updates V1, T2 updates V2), and finally concurrently commit, neither having seen the update performed by the other.

[Snapshot isolation, Wiki.](https://en.wikipedia.org/wiki/Snapshot_isolation)

# tests

- `ws-plain.go` small dataset of 2 keys. reading 2 keys and making disjoint writes to them with an increment. does not show ws.
- `rs-plain.go` small dataset of 2 keys. 1 tx does prefix reads checking for invariant, another tx commits a change in the middle of the first one. does not show rs due to the: _rpc error: code = Aborted desc = too much contention on these datastore entities. please try again. entity group key: (app=e~test!test, test_read_skew, "x")_.
