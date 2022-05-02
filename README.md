
## Overview

This tool is meant to provide a rough estimate on how fast each Pub/Sub can process messages.

When benchmarking a Pub/Sub Systems, we specifically require two distinct roles ( publishers and subscribers ) as benchmark participants - this repo contains code to mimic the publisher and subscriber workloads on Pub/Sub systems.


**Current Pub/Sub Systems supported**:
  - Redis Pub/Sub 
  - Redis Sharded Pub/Sub (since Redis >= 7.0)



Several aspects can dictate the overall system performance, like the:
- Payload size (controlled on publisher)
- Number of Pub/Sub channels (controlled on publisher)
- Total message traffic per channel (controlled on publisher)
- Number of subscribers per channel (controlled on subscriber)
- Subscriber distribution per shard and channel (controlled on subscriber)

## Getting started with docker

### subscriber mode


```bash
docker run --network=host codeperf/pubsub-bench:unstable pubsub-bench subscribe
```

### publisher mode
```bash
docker run --network=host codeperf/pubsub-bench:unstable pubsub-bench publish
```


## Getting Started building from source

### Installing
This benchmark go program is **know to be supported for go >= 1.17**. 
The easiest way to get and install the benchmark Go programs is to use `go get` and then `go install`:

```
go get github.com/filipecosta90/pubsub-sub-bench
cd $GOPATH/src/github.com/filipecosta90/pubsub-sub-bench
make
```
