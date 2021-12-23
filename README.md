## `snake_p2p`

A distributed battle-royal snake game for 2 or more people.

### v0.1 roadmap

- [x] Nodes can connect to other nodes and establish a pub/sub network
- [x] Party protocol (PARTY, JOIN messages, peer connect/disconnect handling)
- [x] Snake protocol (synchronized state via complete graph connection ~~and a blockchain history~~)
- [x] Console game implementation (processing of the events published by the protocols and interaction with the user)

### Party protocol sketch

The party protocol requires an established pub/sub network with a topic `PARTY`. Any user that wishes to have a game can either publish a message to the `PARTY` topic describing what kind of game it wants to play or can listen to the messages posted on the `PARTY` topic and connect to the senders.

For simplicity, let a message on the `PARTY` topic only describe the desired number of players (and publsher peer ID/ multiaddr, maybe). \<I need to research it, but maybe it should also send it every N seconds, so that new peers can catch up.\>

If a user wishes to join a party, it establishes a direct communication with the message publisher. The publisher sends a `connected to <addr>` message to the rest of connected peers. They in turn detect that it is the publisher that got a new connection and try to connect to the new peer. The players will connect to the new peer only if the publisher got connected to it. Upon successful connection, they send `connected to <addr>` message to the publisher. The publisher maintains who is connected to who, and when a complete subgraph of peers that includes the publisher is formed, the publisher sends `party formed []<addr>` message. The peers not in that list are notified that they were not chosen for the party to the user, and those that were, notify that they were. The party protocol finishes execution.

So, the "direct communications" that the peers establish between each other and about whose they inform the publisher are using the snake protocol.

Q1. A peer can notify the publisher that it has lost a connection with another player using an `<addr> disconnected` message. What then?

> The publisher updates its view of the party network and does not issue the `party formed` message that contains both peers.

Q2. What if maliscious peer tries to fool party members to connect to another maliscious node?

> Peers will only connect to another peer if `connected to <addr>` message was sent by the publisher, so this scenario is not likely.

Q3. What if after `party formed` message, some party members loose the connection?

> The snake protocol will time out and the game will be finished.

Q4. Using this protocol a peer can try to enroll to several parties. If it got accepted to one, how can it tell other parties that it quits them?

> It closes the connections to the party members that will then send the `<addr> disconnected` message to the publisher. 

### Snake protocol sketch

To be written...
