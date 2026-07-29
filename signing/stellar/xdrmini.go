package stellar

import "encoding/binary"

type xw struct{ b []byte }

func (w *xw) u32(v uint32) {
	var t [4]byte
	binary.BigEndian.PutUint32(t[:], v)
	w.b = append(w.b, t[:]...)
}

func (w *xw) u64(v uint64) {
	var t [8]byte
	binary.BigEndian.PutUint64(t[:], v)
	w.b = append(w.b, t[:]...)
}

func (w *xw) i64(v int64) { w.u64(uint64(v)) }

func (w *xw) raw(b []byte) { w.b = append(w.b, b...) }

func (w *xw) opaque(b []byte) {
	w.u32(uint32(len(b)))
	w.b = append(w.b, b...)
	for len(w.b)%4 != 0 {
		w.b = append(w.b, 0)
	}
}

type stellarOp struct {
	isPayment bool
	destKey   []byte
	amount    int64
}

type stellarTx struct {
	sourceKey []byte
	fee       uint32
	seqNum    int64
	memo      string
	op        stellarOp
}

func (t stellarTx) marshalTx() []byte {
	w := &xw{}
	w.u32(0)
	w.raw(t.sourceKey)
	w.u32(t.fee)
	w.i64(t.seqNum)
	w.u32(1)
	w.u64(0)
	w.u64(0)
	w.u32(1)
	w.opaque([]byte(t.memo))
	w.u32(1)
	w.u32(0)
	if t.op.isPayment {
		w.u32(1)
		w.u32(0)
		w.raw(t.op.destKey)
		w.u32(0)
		w.i64(t.op.amount)
	} else {
		w.u32(0)
		w.u32(0)
		w.raw(t.op.destKey)
		w.i64(t.op.amount)
	}
	w.u32(0)
	return w.b
}

func (t stellarTx) marshalEnvelope(hints [][4]byte, sigs [][]byte) []byte {
	w := &xw{}
	w.u32(2)
	w.raw(t.marshalTx())
	w.u32(uint32(len(sigs)))
	for i := range sigs {
		w.raw(hints[i][:])
		w.opaque(sigs[i])
	}
	return w.b
}

func (t stellarTx) marshalSigPayload(networkID [32]byte) []byte {
	w := &xw{}
	w.raw(networkID[:])
	w.u32(2)
	w.raw(t.marshalTx())
	return w.b
}
