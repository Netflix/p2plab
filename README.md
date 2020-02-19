# p2plab

[![Build Status](https://travis-ci.com/Netflix/p2plab.svg?branch=master)](https://travis-ci.com/Netflix/p2plab)
[![GoDoc](https://godoc.org/github.com/Netflix/p2plab?status.svg)](https://godoc.org/github.com/Netflix/p2plab)

`p2plab` is infrastructure to benchmark IPFS throughput in reproducible and quantifiable way.

[![asciicast](https://asciinema.org/a/264008.svg)](https://asciinema.org/a/264008)

Key features:

- IPFS infrastructure as code
- Cluster-agnostic benchmarking scenarios
- Live update IPFS infrastructure to different commit
- Distributed tracing

## Getting started

By default, `p2plab` runs with a in-memory driver and can deploy a cluster of IPFS nodes as subprocesses.

First, compile and run `labd`, the main daemon orchestrating `p2plab`:
```sh
export GO111MODULE=on
go get -u github.com/Netflix/p2plab/cmd/labd
labd
```

In a new terminal, compile `labctl`, the CLI to manage the infrastructure and run benchmarks:
```sh
export GO111MODULE=on
go get -u github.com/Netflix/p2plab/cmd/labctl
```

Now you can create your first local cluster using one of the examples:
```sh
$ labctl cluster create --definition ./examples/cluster/same-region.json my-cluster
6:52PM INF Creating node group name=my-cluster
6:52PM INF Updating metadata with new nodes name=my-cluster
6:52PM INF Waiting for healthy nodes name=my-cluster
6:52PM INF Updating cluster metadata name=my-cluster
6:52PM INF Created cluster "my-cluster"
my-cluster

$ labctl node ls my-cluster
+----------------------+-----------+--------------+---------------------------------------------------+----------------+----------------+
|          ID          |  ADDRESS  | GITREFERENCE |                      LABELS                       |   CREATEDAT    |   UPDATEDAT    |
+----------------------+-----------+--------------+---------------------------------------------------+----------------+----------------+
| bp3ept7ic6vdctur3dag | 127.0.0.1 | HEAD         | bp3ept7ic6vdctur3dag,t2.micro,us-west-2           | 20 seconds ago | 20 seconds ago |
| bp3eptfic6vdctur3db0 | 127.0.0.1 | HEAD         | bp3eptfic6vdctur3db0,neighbors,t2.micro,us-west-2 | 20 seconds ago | 20 seconds ago |
| bp3eptvic6vdctur3dbg | 127.0.0.1 | HEAD         | bp3eptvic6vdctur3dbg,neighbors,t2.micro,us-west-2 | 20 seconds ago | 20 seconds ago |
+----------------------+-----------+--------------+---------------------------------------------------+----------------+----------------+
```

In the labels column, notice how two out of three of the nodes have the label `neighbors`. This will come in handy when running our benchmark.
Benchmarks are executed from scenarios, and scenarios are decoupled from the cluster we benchmark because they operate on labels. Whether we're running in `us-west-1` or `us-east-2`, or our cluster has 3 or 50 nodes, you can still execute the same scenario given that the appropriate nodes are labelled.

Let's create our first scenario using one of the examples:
```sh
$ labctl scenario create ./examples/scenario/neighbors.json
6:59PM INF Created scenario "neighbors"
neighbors

$ labctl scenario inspect neighbors
{
    "ID": "neighbors",
    "Definition": {
        "objects": {
            "golang": {
                "type": "oci",
                "source": "docker.io/library/golang:latest",
                "layout": "",
                "chunker": "",
                "rawLeaves": false,
                "hashFunc": "",
                "maxLinks": 0
            }
        },
        "seed": {
            "neighbors": "golang"
        },
        "benchmark": {
            "(not 'neighbors')": "golang"
        }
    },
    "Labels": [
        "neighbors"
    ],
    "CreatedAt": "2020-02-14T18:59:44.698063252Z",
    "UpdatedAt": "2020-02-14T18:59:44.698063252Z"
}
```

When we finally run our benchmark, `labd` will download the objects in the scenario, in this case the `golang` OCI image and convert it into a IPFS DAG. Then it will follow the `seed` stage and distribute the object `golang` to nodes matching the label `neighbors`. The benchmark will then measure how long it takes for nodes that **don't** match the label `neighbors` with the object `golang`.

```sh
$ labctl benchmark create my-cluster neighbors
7:02PM INF Retrieving nodes in cluster bid=my-cluster-neighbors-1581706936119660719
7:02PM INF Resolving git references bid=my-cluster-neighbors-1581706936119660719
7:02PM INF Building p2p app(s) bid=my-cluster-neighbors-1581706936119660719 commits=["5f7c8e0d9104c76974db9640c05beec429f56e36"]
7:02PM INF Updating cluster bid=my-cluster-neighbors-1581706936119660719
7:02PM INF Retrieving peer infos bid=my-cluster-neighbors-1581706936119660719
7:02PM INF Connecting cluster bid=my-cluster-neighbors-1581706936119660719
7:02PM INF Creating scenario plan bid=my-cluster-neighbors-1581706936119660719
7:02PM INF Transforming objects into IPLD DAGs bid=my-cluster-neighbors-1581706936119660719
7:02PM INF Resolving OCI reference bid=my-cluster-neighbors-1581706936119660719 source=docker.io/library/golang:latest
7:02PM INF Resolved reference to digest bid=my-cluster-neighbors-1581706936119660719 digest=sha256:9295ba678e3764d79ac0aeabdbcf281a91933c81c8de29387d8a2f557e256cdb source=docker.io/library/golang:latest
7:02PM INF Converting manifest recursively to IPLD DAG bid=my-cluster-neighbors-1581706936119660719 digest=sha256:9295ba678e3764d79ac0aeabdbcf281a91933c81c8de29387d8a2f557e256cdb
7:03PM INF Constructing Unixfs directory over manifest blobs bid=my-cluster-neighbors-1581706936119660719 target=sha256:29c7ea58b504cee59a6f4e442867151f0763be246d5c9d06f499ac841118f93f
7:03PM INF Planning scenario seed bid=my-cluster-neighbors-1581706936119660719
7:03PM INF Planning scenario benchmark bid=my-cluster-neighbors-1581706936119660719
7:03PM INF Creating benchmark metadata bid=my-cluster-neighbors-1581706936119660719
7:03PM INF Executing scenario plan bid=my-cluster-neighbors-1581706936119660719
7:03PM INF Seeding cluster bid=my-cluster-neighbors-1581706936119660719
7:03PM INF Seeding completed bid=my-cluster-neighbors-1581706936119660719
7:03PM INF Starting a session for benchmarking bid=my-cluster-neighbors-1581706936119660719
7:03PM INF Waiting for healthy nodes bid=my-cluster-neighbors-1581706936119660719
7:03PM INF Retrieving peer infos bid=my-cluster-neighbors-1581706936119660719
7:03PM INF Connecting cluster bid=my-cluster-neighbors-1581706936119660719
7:03PM INF Benchmarking cluster bid=my-cluster-neighbors-1581706936119660719
7:03PM INF Benchmark completed bid=my-cluster-neighbors-1581706936119660719
7:03PM INF Retrieving reports bid=my-cluster-neighbors-1581706936119660719
7:03PM INF Ending the session bid=my-cluster-neighbors-1581706936119660719
7:03PM INF Updating benchmark metadata bid=my-cluster-neighbors-1581706936119660719
7:03PM INF Completed benchmark "my-cluster-neighbors-1581706936119660719"
# Summary
Total time: 4 seconds 971 milliseconds
Trace:

# Bandwidth
+-------------------+----------------------+---------+----------+----------+----------+
|       QUERY       |         NODE         | TOTALIN | TOTALOUT |  RATEIN  | RATEOUT  |
+-------------------+----------------------+---------+----------+----------+----------+
| (not 'neighbors') | bp3ept7ic6vdctur3dag | 397 MB  |  204 kB  | 106 MB/s | 54 kB/s  |
+-------------------+----------------------+---------+----------+----------+----------+
|         -         | bp3eptfic6vdctur3db0 |  96 kB  |  188 MB  | 26 kB/s  | 52 MB/s  |
+                   +----------------------+---------+----------+----------+----------+
|                   | bp3eptvic6vdctur3dbg | 109 kB  |  212 MB  | 28 kB/s  | 55 MB/s  |
+-------------------+----------------------+---------+----------+----------+----------+
|                            TOTAL         | 397 MB  |  400 MB  | 106 MB/s | 107 MB/s |
+-------------------+----------------------+---------+----------+----------+----------+

# Bitswap
+-------------------+----------------------+------------+------------+-----------+----------+----------+---------+
|       QUERY       |         NODE         | BLOCKSRECV | BLOCKSSENT | DUPBLOCKS | DATARECV | DATASENT | DUPDATA |
+-------------------+----------------------+------------+------------+-----------+----------+----------+---------+
| (not 'neighbors') | bp3ept7ic6vdctur3dag |   1,866    |     0      |     2     |  481 MB  |   0 B    | 263 kB  |
+-------------------+----------------------+------------+------------+-----------+----------+----------+---------+
|         -         | bp3eptfic6vdctur3db0 |     0      |    888     |     0     |   0 B    |  228 MB  |   0 B   |
+                   +----------------------+            +------------+           +          +----------+         +
|                   | bp3eptvic6vdctur3dbg |            |    978     |           |          |  252 MB  |         |
+-------------------+----------------------+------------+------------+-----------+----------+----------+---------+
|                            TOTAL         |   1,866    |   1,866    |     2     |  481 MB  |  481 MB  | 263 kB  |
+-------------------+----------------------+------------+------------+-----------+----------+----------+---------+
```

Well done! You've ran your first benchmark and transferred a container image over IPFS.
