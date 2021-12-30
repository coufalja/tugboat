/*
Package tugboat is a feature complete and highly optimized multi-group Raft
implementation for providing consensus in distributed systems.

The NodeHost struct is the facade interface for all features provided by the
tugboat package. Each NodeHost instance usually runs on a separate server
managing CPU, storage and network resources used for achieving consensus. Each
NodeHost manages Raft nodes from different Raft groups known as Raft clusters.
Each Raft cluster is identified by its ClusterID, it usually consists of
multiple nodes (also known as replicas) each identified by a NodeID value.
Nodes from the same Raft cluster suppose to be distributed on different NodeHost
instances across the network, this brings fault tolerance for machine and
network failures as application data stored in the Raft cluster will be
available as long as the majority of its managing NodeHost instances (i.e. its
underlying servers) are accessible.

Arbitrary number of Raft clusters can be launched across the network to
aggregate distributed processing and storage capacities. Users can also make
membership change requests to add or remove nodes from selected Raft cluster.

User applications can leverage the power of the Raft protocol by implementing
the IStateMachine or IOnDiskStateMachine component, as defined in
github.com/coufalja/tugboat/statemachine. Known as user state machines, each
IStateMachine or IOnDiskStateMachine instance is in charge of updating, querying
and snapshotting application data with minimum exposure to the Raft protocol
itself.

Tugboat guarantees the linearizability of your I/O when interacting with the
IStateMachine or IOnDiskStateMachine instances. In plain English, writes (via
making proposals) to your Raft cluster appears to be instantaneous, once a write
is completed, all later reads (via linearizable read based on Raft's ReadIndex
protocol) should return the value of that write or a later write. Once a value
is returned by a linearizable read, all later reads should return the same value
or the result of a later write.

To strictly provide such guarantee, we need to implement the at-most-once
semantic. For a client, when it retries the proposal that failed to complete by
its deadline, it faces the risk of having the same proposal committed and
applied twice into the user state machine. Tugboat prevents this by
implementing the client session concept described in Diego Ongaro's PhD thesis.
*/
package tugboat
