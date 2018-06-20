using Go = import "/github.com/glycerine/go-capnproto/go.capnp";
$Go.package("goserbench");
$Go.import("github.com/alecthomas/go_serialization_benchmarks");

@0x99ea7c74456111bd;

struct CapnpA {
	ID        @0 :Int64;
	Tag       @1 :Uint8;
	Timestamp @2 :Int64;
	Frames    @3 :List(Uint64);
	GID       @4 :Int64;
	TxID      @5 :Uint64;
}
