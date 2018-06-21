using Go = import "/github.com/glycerine/go-capnproto/go.capnp";
$Go.package("goserbench");
$Go.import("bitbucket.org/yuuki0xff/goapptrace-codec-benchmarks");

@0x99ea7c74456111bd;

struct CapnpA {
	id        @0 :Int64;
	tag       @1 :UInt8;
	timestamp @2 :Int64;
	frames    @3 :List(UInt64);
	gid       @4 :Int64;
	txid      @5 :UInt64;
}
