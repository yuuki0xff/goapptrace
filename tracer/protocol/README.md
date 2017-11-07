# Protocol Specification Version 1.0 (Draft)
## Summary
This binary protocol is for the purpose of communication with goapptrace server and trace target process.
All data can split to unit called the _packet_.
This protocol exchange _packet_'s through TCP.
Unix socket supports is planned, but it is not supported currently.

## Packet Specification
Packet is unit of encode/decode.
Packet MUST encode to binary by the gob encoder.
Packet can classable to _HelloPacket_ type, _HeaderPacket_ type and _DataPacket_ type.

* _HelloPacket_ only use when negotiation process.
  That is two types of `ClientHelloPacket` and `ServerHelloPacket`.
  Those packet is for the purpose of sends the protocol version etc. to the partner.
* `HeaderPacket` notify about a _DataPacket_ of packet type and packet length.
  Client and Server **MUST** send that packet before each data packet sends.
* _DataPacket_ is ... see the source code.

For more information about those packet fields, see the source code in `packet.go`.
For usage, see _Protocol Negotiation Sequence_ section.

## Protocol Version Specification
Protocol version MUST have only major version and minor version.
String expression of protocol version is major version and minor version splitted by "." like  "[major].[minor]". 

Example: When major version is 1 and minor version is 23, String expression is "1.23"

## Protocol Negotiation Sequence
1. Client open TCP socket with the server.
2. Client sends a packet of ClientHelloPacket to the server.
   Server receives a packet from client, and checks protocol version.
   If any errors occurs, server can close TCP socket immediately.
3. Server sends a packet of ServerHelloPacket to the server.
   Client receives a packet from server, and checks checks it.
   If any errors occurs, client can close TCP socket immediately.
4. The negotiation process will be succeeded.
   Client and Server are sends any packet to the partner any time.
5. If Client/Server want to close this TCP session, SHOULD send a ShutdownPacket to a partner before close this TCP session.

```text
          [Negotiation Flow]
Client                          Server

  +                                +
  |                                |
  |   [2] ClientHelloPacket        |
  |                                |
  +----------------------------->  |
  |                                |
  |   [3] ServerHelloPacket        |
  |                                |
  |   <----------------------------+
  |                                |
  |   (Negotiation Complete)       |
  |   [4] HeaderPacket/DataPacket  |
  |   can send any time.           |
  |                                |
  |   <------------------------->  |
  |                                |
  |   [5] ShutdownPacket           |
  |   (EX: close from client)      |
  |                                |
  +----------------------------->  |
  |                                |
  |   [6] We SHOULD close this     |
  +   TCP session.                 +
```

