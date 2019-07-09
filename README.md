* Bitmarkd Broadcast Monitor

This repositor is used to monitor bitmarkd broadcast drop rate, more specifically, using block broadcast and heartbeat as metrics.

There exist two kinds of communication, one is periodic and the other is random:

1. Periodic communication

Every 30 seconds, bitmarkd broadcast heartbeat signal to connected clients, so for a period of time, expected receive count can be calculated. Use this expected value as denominator, actual received haertbeat count as nominator, drop rate can be calculated.

1. Random communication

Whenever bitmarkd receives a block, it will broadcast this block to connected clients, since when will a block generated is random, the expected number of block broadcast can not directly calculated by some formula. But since blockchain will eventually grow larger, it is possible to use block height as an indicator to calculate expected block broadcast number.

For example, if bitmarkd is now at block height 100, and monitor service receives block broadcast of block height 101, 102, 103. For some interval, monitor service ask bitmarkd the highest block number, assume it's 104 and not fork happens. Since bitmarkd should broadcast every incoming block then the drop rate is 25% (3 block is received - 101, 102, 103, and expected 4 recieved - 101, 102, 103, 104).

If there's a fork, then expected value and received value will be increase, using above example, if this bitmarkd grows from block 100 to block 103, and at block 103 it founds a fork and delete existing block to 101, resync block 102 and 103, then theoretically, monitor service should receive broadcast of 101, 102, 103, 102', 103'. The first pair of 102 and 103 are original blocks, the latter pair of 102 and 103 means fork happens and receive new blocks of 102' and 103'.
