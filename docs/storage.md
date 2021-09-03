# Raft Log Storage #

Dragonboat uses [Pebble](https://github.com/cockroachdb/pebble) to store Raft logs.

## Compatibility ##

Pebble is a new Key-Value store implemented in Go with bidirectional compatibility with RocksDB.

## Pebble ##

Pebble is used by default, no configuration is required.

## Use custom storage solution ##

You can extend Dragonboat to use your preferred storage solution to store Raft logs -

* implement the ILogDB interface defined in the github.com/coufalja/tugboat/raftio package
* pass a factory function that creates such a custom Log DB instance to the LogDBFactory field of your NodeHostConfig.Expert instance
