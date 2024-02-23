--- 
order: 14
---

# Proposer-Based Timestamp (PBTS)

This document overviews the Proposer-Based Timestamp (PBTS)
algorithm introduced in CometBFT version v1.0.
It outlines the core functionality of the algorithm and details the consensus
parameters that govern its operation.

## Overview 

The PBTS algorithm defines a way for a blockchain to create block
timestamps that are within a reasonable bound of the clocks of the validators on
the network. 
It replaces the BFT Time algorithm for timestamp assignment, that computes the
timestamp of a block using the timestamps included in precommit messages.

### Block Timestamps

Each block produced by CometBFT contains a timestamp, represented by the `Time`
field of the block's `Header`.

The timestamp of each block is expected to be a meaningful representation of time that is
useful for the protocols and applications built on top of CometBFT.
The following protocols and application features require a reliable source of time:

* Light Clients [rely on correspondence between their known time][light-client-verification] and the block time for block verification.
* Evidence expiration is determined [either in terms of heights or in terms of time][evidence-verification].
* Unbonding of staked assets in the Cosmos Hub [occurs after a period of 21
 days](https://github.com/cosmos/governance/blob/master/params-change/Staking.md#unbondingtime).
* IBC packets can use either a [timestamp or a height to timeout packet
 delivery](https://ibc.cosmos.network/v8/ibc/light-clients/updates-and-misbehaviour?_highlight=time#checkformisbehaviour).

### Enabling PBTS

The PBTS algorithm is **not enabled by default** in CometBFT v1.0.

If a network upgrades to CometBFT v1.0, it will still use the BFT Time
algorithm until PBTS is enabled.
The same applies to new networks that do not change the default values for
consensus parameters in the genesis file.

Enabling PBTS requires configuring the [consensus parameters](#consensus-parameters)
that govern the operation of the algorithm.

### Selecting a Timestamp

When a validator creates a new block, it reads the time from its local clock
and uses this reading as the timestamp for the block.
The proposer of a block is thus free to select the block timestamp, but this
timestamp have to be validated by other nodes in the network.

### Validating Timestamps

When each validator on the network receives a proposed block, it performs a
series of checks to ensure that the block can be considered valid as a
candidate to be the next block in the chain.
If the block is considered invalid, the validator issues a `nil` prevote,
signaling to the rest of the network that the proposed block is not valid.

The PBTS algorithm performs a validity check on the timestamp of proposed
blocks. When a validator receives a proposal it ensures that the timestamp in
the proposal is within a bound of the validator's local clock.
Specifically, the algorithm checks that the timestamp is
no more than `Precision` greater than the node's local clock
(i.e., not in the future)
and no less than `MessageDelay + Precision` behind than the node's local clock
(i.e., not too far in the past).
`Precision` and `MessageDelay` are consensus parameters, 
and are the same across all nodes.

If the proposed block's timestamp is within the window of acceptable
timestamps, the timestamp is considered **timely**.
If the block timestamp is **not timely**, the validator rejects the block by
issuing a `nil` prevote.

### Clock Synchronization

The PBTS algorithm requires the clocks of the validators in the network to be
within `Precision` of each other. In practice, this means that validators
should periodically synchronize their clocks to a reliable NTP server.
Validators whose clocks drift too far away from the rest of the network will no
longer propose blocks with valid timestamps. Additionally, they will not consider
the timestamps of blocks proposed by their peers to be valid either.


## Consensus Parameters

The functionality of the PBTS algorithm is governed by two consensus
parameters: the synchronous parameters `Precision` and `MessageDelay`.
An additional consensus parameter `PbtsEnableHeight` is used to enable PBTS
when instantiating a new network or when upgrading an existing a network that
uses BFT Time.

Consensus parameters are configured by the ABCI application and are the same
across all nodes in the network.

### `SynchronyParams.Precision`

The `Precision` parameter configures the acceptable upper-bound of clock drift
among all of the validators in the network.
Any two validators are expected to have clocks that differ by at most
`Precision` at any given instant.

Networks should choose a `Precision` that is large enough to represent the
worst-case for the clock drift among all participants.
Due to the [leap second events](https://github.com/tendermint/tendermint/issues/7724),
it is recommended to use set `Precision` to at least `500ms`.

The `Precision` parameter is given in milliseconds.

### `SynchronyParams.MessageDelay`

The `MessageDelay` parameter configures the acceptable upper-bound for the
end-to-end delay for transmitting a `Proposal` message from the proposer to
_all_ validators in the network.

Networks should choose a `MessageDelay` that is large enough to represent the
worst-case delay for a message to reach all participants.
This delay does not depend, a priori, on the size of proposed blocks, since it
applies to the `Proposal` messages.
It does depend on the number of participants and latency of their connections.

The `MessageDelay` parameter is given in milliseconds.

### `FeatureParams.PbtsEnableHeight`

The `PbtsEnableHeight` parameter configures the first height at which the PBTS
algorithm should be adopted for generating and validating block timestamps in a network.
In heights up to `PbtsEnableHeight - 1`, the generation and validation of block
timestamps in the network will be carried out using the legacy BFT Time algorithm.

While `PbtsEnableHeight` is set to `0`, the network will adopt the legacy BFT
Time algorithm.

When `PbtsEnableHeight` is set to a height `H > 0`, the network will switch to
the PBTS algorithm from height `H`.
The network will still adopt the legacy BFT Time algorithm to produce and
validate block timestamps until height `H - 1`.
The enable height `H` must be a future height, i.e., larger than the current
blockchain height.

Once `PbtsEnableHeight` is set and the PBTS algorithm is enabled (i.e., from height
`PbtsEnableHeight`), it is not possible to return to the legacy BFT Time algorithm.
The switch to PBTS is therefore irreversible.

Finally, if `PbtsEnableHeight` is set `1` in the genesis file or by the application
upon the ABCI `InitChain` method, the network will adopt PBTS from the first
height. This is the recommended setup for new chains.

The `PbtsEnableHeight` parameter is an integer.

## Important Notes

When configuring a network to adopt the PBTS algorithm, the follow steps must be considered:

1. Make sure that the clocks of validators are [synchronized](#clock-synchronization) **before** enabling PBTS.
1. Make sure that the configured value for [`SynchronyParams.Precision`](#synchronyparamsprecision) is
   reasonable.
1. Make sure that the configured value for [`SynchronyParams.MessageDelay`](#synchronyparamsmessagedelay) is
   reasonable and large enough to reflect the worst-case delay for messages in the network.
   Setting this parameter to a small value may impact the progress of the
   network, namely blocks may take very long to be committed.
1. Make sure that the block times **currently** produced by the network do not
   differ too much from real time.

Observation 4. is important because, with the adoption of PBTS, block times are
expected to converge to values that bear resemblance to real time.
At the same time, the same property of monotonic block times guaranteed by BFT
Time is also ensured by PBTS.
This means that proposers using PBTS will **wait** until the time they read
from their local clocks becomes bigger than the time of the last committed
block before proposing a new block.

As a result, if the time of the last block produced using BFT Time is far in
the future, then the first block produced using PBTS will take very long to be
committed: the time it takes for the clock of the proposer to reach the time of
the previously committed block.
To prevent this from happening, first, follow recommendation 1., i.e., synchronize
the validators' clocks.
Then wait until the block times produced by BFT Time converge to values that do
not differ too much from real time (observation 4.).
This may take a long time, because in BFT Time if the value a validator reads
from its local clock is smaller than the time of the previous block, then the
time it sets to a new block will be the time of the previous block plus `1ms`.
It may take a while, but blocks times eventually converge to real time.

## See Also

* [Block Time specification][block-time-spec]: overview of block timestamps properties.
* [Consensus parameters][consensus-parameters]: list of consensus parameters, their usage and validation.
* [PBTS specification][pbts-spec]: formal specification and all of the details of the PBTS algorithm.
* [BFT Time specification][bft-time-spec]: all details of the legacy BFT Time algorithm to compute block times.
* [Proposer-Based Timestamps Runbook][pbts-runbook]: a guide for diagnosing and
  fix issues related to clock synchronization and the configuration of the
  `SynchronyParams` consensus parameters adopted by PBTS.

[pbts-spec]: https://github.com/cometbft/cometbft/blob/main/spec/consensus/proposer-based-timestamp/README.md
[bft-time-spec]: https://github.com/cometbft/cometbft/blob/main/spec/consensus/bft-time.md
[block-time-spec]: https://github.com/cometbft/cometbft/blob/main/spec/consensus/time.md
[pbts-runbook]: ../../guides/tools/proposer-based-timestamps-runbook.md

[consensus-parameters]: https://github.com/cometbft/cometbft/blob/main/spec/abci/abci%2B%2B_app_requirements.md#consensus-parameters

[light-client-verification]: https://github.com/cometbft/cometbft/blob/main/spec/light-client/verification/README.md#failure-model
[evidence-verification]: https://github.com/cometbft/cometbft/blob/main/spec/consensus/evidence.md#verification