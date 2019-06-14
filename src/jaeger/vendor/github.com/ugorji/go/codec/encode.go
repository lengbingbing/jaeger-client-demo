// Copyright (c) 2012-2018 Ugorji Nwoke. All rights reserved.
// Use of this source code is governed by a MIT license found in the LICENSE file.

package codec

import (
	"encoding"
	"errors"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"time"
)

// defEncByteBufSize is the default size of []byte used
// for bufio buffer or []byte (when nil passed)
const defEncByteBufSize = 1 << 10 // 4:16, 6:64, 8:256, 10:1024

var errEncoderNotInitialized = errors.New("Encoder not initialized")

// encDriver abstracts the actual codec (binc vs msgpack, etc)
type encDriver interface {
	EncodeNil()
	EncodeInt(i int64)
	EncodeUint(i uint64)
	EncodeBool(b bool)
	EncodeFloat32(f float32)
	EncodeFloat64(f float64)
	// encodeExtPreamble(xtag byte, length int)
	EncodeRawExt(re *RawExt)
	EncodeExt(v interface{}, xtag uint64, ext Ext)
	EncodeStringEnc(c charEncoding, v string) // c cannot be cRAW
	// EncodeSymbol(v string)
	EncodeStringBytesRaw(v []byte)
	EncodeTime(time.Time)
	//encBignum(f *big.Int)
	//encStringRunes(c charEncoding, v []rune)
	WriteArrayStart(length int)
	WriteArrayEnd()
	WriteMapStart(length int)
	WriteMapEnd()

	reset()
	atEndOfEncode()
}

type encDriverContainerTracker interface {
	WriteArrayElem()
	WriteMapElemKey()
	WriteMapElemValue()
}

type encDriverAsis interface {
	EncodeAsis(v []byte)
}

type encodeError struct {
	codecError
}

func (e encodeError) Error() string {
	return fmt.Sprintf("%s encode error: %v", e.name, e.err)
}

type encDriverNoopContainerWriter struct{}

func (encDriverNoopContainerWriter) WriteArrayStart(length int) {}
func (encDriverNoopContainerWriter) WriteArrayEnd()             {}
func (encDriverNoopContainerWriter) WriteMapStart(length int)   {}
func (encDriverNoopContainerWriter) WriteMapEnd()               {}
func (encDriverNoopContainerWriter) atEndOfEncode()             {}

// func (encDriverNoopContainerWriter) WriteArrayElem()            {}
// func (encDriverNoopContainerWriter) WriteMapElemKey()           {}
// func (encDriverNoopContainerWriter) WriteMapElemValue()         {}

// type encDriverTrackContainerWriter struct {
// 	c containerState
// }

// func (e *encDriverTrackContainerWriter) WriteArrayStart(length int) { e.c = containerArrayStart }
// func (e *encDriverTrackContainerWriter) WriteArrayElem()            { e.c = containerArrayElem }
// func (e *encDriverTrackContainerWriter) WriteArrayEnd()             { e.c = containerArrayEnd }
// func (e *encDriverTrackContainerWriter) WriteMapStart(length int)   { e.c = containerMapStart }
// func (e *encDriverTrackContainerWriter) WriteMapElemKey()           { e.c = containerMapKey }
// func (e *encDriverTrackContainerWriter) WriteMapElemValue()         { e.c = containerMapValue }
// func (e *encDriverTrackContainerWriter) WriteMapEnd()               { e.c = containerMapEnd }
// func (e *encDriverTrackContainerWriter) atEndOfEncode()             {}

// EncodeOptions captures configuration options during encode.
type EncodeOptions struct {
	// WriterBufferSize is the size of the buffer used when writing.
	//
	// if > 0, we use a smart buffer internally for performance purposes.
	WriterBufferSize int

	// ChanRecvTimeout is the timeout used when selecting from a chan.
	//
	// Configuring this controls how we receive from a chan during the encoding process.
	//   - If ==0, we only consume the elements currently available in the chan.
	//   - if  <0, we consume until the chan is closed.
	//   - If  >0, we consume until this timeout.
	ChanRecvTimeout time.Duration

	// StructToArray specifies to encode a struct as an array, and not as a map
	StructToArray bool

	// Canonical representation means that encoding a value will always result in the same
	// sequence of bytes.
	//
	// This only affects maps, as the iteration order for maps is random.
	//
	// The implementation MAY use the natural sort order for the map keys if possible:
	//
	//     - If there is a natural sort order (ie for number, bool, string or []byte keys),
	//       then the map keys are first sorted in natural order and then written
	//       with corresponding map values to the strema.
	//     - If there is no natural sort order, then the map keys will first be
	//       encoded into []byte, and then sorted,
	//       before writing the sorted keys and the corresponding map values to the stream.
	//
	Canonical bool

	// CheckCircularRef controls whether we check for circular references
	// and error fast during an encode.
	//
	// If enabled, an error is received if a pointer to a struct
	// references itself either directly or through one of its fields (iteratively).
	//
	// This is opt-in, as there may be a performance hit to checking circular references.
	CheckCircularRef bool

	// RecursiveEmptyCheck controls whether we descend into interfaces, structs and pointers
	// when checking if a value is empty.
	//
	// Note that this may make OmitEmpty more expensive, as it incurs a lot more reflect calls.
	RecursiveEmptyCheck bool

	// Raw controls whether we encode Raw values.
	// This is a "dangerous" option and must be explicitly set.
	// If set, we blindly encode Raw values as-is, without checking
	// if they are a correct representation of a value in that format.
	// If unset, we error out.
	Raw bool

	// StringToRaw controls how strings are encoded.
	//
	// As a go string is just an (immutable) sequence of bytes,
	// it can be encoded either as raw bytes or as a UTF string.
	//
	// By default, strings are encoded as UTF-8.
	// but can be treated as []byte during an encode.
	//
	// Note that things which we know (by definition) to be UTF-8
	// are ALWAYS encoded as UTF-8 strings.
	// These include encoding.TextMarshaler, time.Format calls, struct field names, etc.
	StringToRaw bool

	// // AsSymbols defines what should be encoded as symbols.
	// //
	// // Encoding as symbols can reduce the encoded size significantly.
	// //
	// // However, during decoding, each string to be encoded as a symbol must
	// // be checked to see if it has been seen before. Consequently, encoding time
	// // will increase if using symbols, because string comparisons has a clear cost.
	// //
	// // Sample values:
	// //   AsSymbolNone
	// //   AsSymbolAll
	// //   AsSymbolMapStringKeys
	// //   AsSymbolMapStringKeysFlag | AsSymbolStructFieldNameFlag
	// AsSymbols AsSymbolFlag
}

// ---------------------------------------------

func (e *Encoder) rawExt(f *codecFnInfo, rv reflect.Value) {
	e.e.EncodeRawExt(rv2i(rv).(*RawExt))
}

func (e *Encoder) ext(f *codecFnInfo, rv reflect.Value) {
	e.e.EncodeExt(rv2i(rv), f.xfTag, f.xfFn)
}

func (e *Encoder) selferMarshal(f *codecFnInfo, rv reflect.Value) {
	rv2i(rv).(Selfer).CodecEncodeSelf(e)
}

func (e *Encoder) binaryMarshal(f *codecFnInfo, rv reflect.Value) {
	bs, fnerr := rv2i(rv).(encoding.BinaryMarshaler).MarshalBinary()
	e.marshalRaw(bs, fnerr)
}

func (e *Encoder) textMarshal(f *codecFnInfo, rv reflect.Value) {
	bs, fnerr := rv2i(rv).(encoding.TextMarshaler).MarshalText()
	e.marshalUtf8(bs, fnerr)
}

func (e *Encoder) jsonMarshal(f *codecFnInfo, rv reflect.Value) {
	bs, fnerr := rv2i(rv).(jsonMarshaler).MarshalJSON()
	e.marshalAsis(bs, fnerr)
}

func (e *Encoder) raw(f *codecFnInfo, rv reflect.Value) {
	e.rawBytes(rv2i(rv).(Raw))
}

func (e *Encoder) kInvalid(f *codecFnInfo, rv reflect.Value) {
	e.e.EncodeNil()
}

func (e *Encoder) kErr(f *codecFnInfo, rv reflect.Value) {
	e.errorf("unsupported kind %s, for %#v", rv.Kind(), rv)
}

func (e *Encoder) kSlice(f *codecFnInfo, rv reflect.Value) {
	// array may be non-addressable, so we have to manage with care
	//   (don't call rv.Bytes, rv.Slice, etc).
	// E.g. type struct S{B [2]byte};
	//   Encode(S{}) will bomb on "panic: slice of unaddressable array".
	if f.seq != seqTypeArray {
		if rvisnil(rv) {
			e.e.EncodeNil()
			return
		}
		// If in this method, then there was no extension function defined.
		// So it's okay to treat as []byte.
		if f.ti.rtid == uint8SliceTypId {
			e.e.EncodeStringBytesRaw(rv.Bytes())
			return
		}
	}
	if f.seq == seqTypeChan && f.ti.chandir&uint8(reflect.RecvDir) == 0 {
		e.errorf("send-only channel cannot be encoded")
	}
	mbs := f.ti.mbs
	rtelem := f.ti.elem

	// if a slice, array or chan of bytes, treat specially
	if !mbs && uint8TypId == rt2id(rtelem) { // NOT rtelem.Kind() == reflect.Uint8
		e.kSliceBytes(rv, f.seq)
		return
	}

	// if chan, consume chan into a slice, and work off that slice.
	if f.seq == seqTypeChan {
		rvcs := reflect.Zero(reflect.SliceOf(rtelem))
		timeout := e.h.ChanRecvTimeout
		if timeout < 0 { // consume until close
			for {
				recv, recvOk := rv.Recv()
				if !recvOk {
					break
				}
				rvcs = reflect.Append(rvcs, recv)
			}
		} else {
			cases := make([]reflect.SelectCase, 2)
			cases[0] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: rv}
			if timeout == 0 {
				cases[1] = reflect.SelectCase{Dir: reflect.SelectDefault}
			} else {
				tt := time.NewTimer(timeout)
				cases[1] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(tt.C)}
			}
			for {
				chosen, recv, recvOk := reflect.Select(cases)
				if chosen == 1 || !recvOk {
					break
				}
				rvcs = reflect.Append(rvcs, recv)
			}
		}
		rv = rvcs // TODO: ensure this doesn't mess up anywhere that rv of kind chan is expected
	}

	var l = rv.Len() // rv may be slice or array
	if mbs {
		if l%2 == 1 {
			e.errorf("mapBySlice requires even slice length, but got %v", l)
			return
		}
		e.mapStart(l / 2)
	} else {
		e.arrayStart(l)
	}

	if l > 0 {
		var fn *codecFn
		for rtelem.Kind() == reflect.Ptr {
			rtelem = rtelem.Elem()
		}
		// if kind is reflect.Interface, do not pre-determine the
		// encoding type, because preEncodeValue may break it down to
		// a concrete type and kInterface will bomb.
		if rtelem.Kind() != reflect.Interface {
			fn = e.h.fn(rtelem)
		}
		for j := 0; j < l; j++ {
			if mbs {
				if j%2 == 0 {
					e.mapElemKey()
				} else {
					e.mapElemValue()
				}
			} else {
				e.arrayElem()
			}
			e.encodeValue(rv.Index(j), fn)
		}
	}

	if mbs {
		e.mapEnd()
	} else {
		e.arrayEnd()
	}
}

func (e *Encoder) kSliceBytes(rv reflect.Value, seq seqType) {
	switch seq {
	case seqTypeSlice:
		e.e.EncodeStringBytesRaw(rv.Bytes())
	case seqTypeArray:
		var l = rv.Len()
		if rv.CanAddr() {
			e.e.EncodeStringBytesRaw(rv.Slice(0, l).Bytes())
		} else {
			var bs []byte
			if l <= cap(e.b) {
				bs = e.b[:l]
			} else {
				bs = make([]byte, l)
			}
			reflect.Copy(reflect.ValueOf(bs), rv)
			e.e.EncodeStringBytesRaw(bs)
		}
	case seqTypeChan:
		// do not use range, so that the number of elements encoded
		// does not change, and encoding does not hang waiting on someone to close chan.
		// for b := range rv2i(rv).(<-chan byte) { bs = append(bs, b) }
		// ch := rv2i(rv).(<-chan byte) // fix error - that this is a chan byte, not a <-chan byte.

		if rvisnil(rv) {
			e.e.EncodeNil()
			break
		}
		bs := e.b[:0]
		irv := rv2i(rv)
		ch, ok := irv.(<-chan byte)
		if !ok {
			ch = irv.(chan byte)
		}

	L1:
		switch timeout := e.h.ChanRecvTimeout; {
		case timeout == 0: // only consume available
			for {
				select {
				case b := <-ch:
					bs = append(bs, b)
				default:
					break L1
				}
			}
		case timeout > 0: // consume until timeout
			tt := time.NewTimer(timeout)
			for {
				select {
				case b := <-ch:
					bs = append(bs, b)
				case <-tt.C:
					// close(tt.C)
					break L1
				}
			}
		default: // consume until close
			for b := range ch {
				bs = append(bs, b)
			}
		}

		e.e.EncodeStringBytesRaw(bs)
	}
}

func (e *Encoder) kStructNoOmitempty(f *codecFnInfo, rv reflect.Value) {
	sfn := structFieldNode{v: rv, update: false}
	if f.ti.toArray || e.h.StructToArray { // toArray
		e.arrayStart(len(f.ti.sfiSrc))
		for _, si := range f.ti.sfiSrc {
			e.arrayElem()
			e.encodeValue(sfn.field(si), nil)
		}
		e.arrayEnd()
	} else {
		e.mapStart(len(f.ti.sfiSort))
		for _, si := range f.ti.sfiSort {
			e.mapElemKey()
			e.kStructFieldKey(f.ti.keyType, si.encNameAsciiAlphaNum, si.encName)
			e.mapElemValue()
			e.encodeValue(sfn.field(si), nil)
		}
		e.mapEnd()
	}
}

func (e *Encoder) kStructFieldKey(keyType valueType, encNameAsciiAlphaNum bool, encName string) {
	encStructFieldKey(encName, e.e, e.w(), keyType, encNameAsciiAlphaNum, e.js)
}

func (e *Encoder) kStruct(f *codecFnInfo, rv reflect.Value) {
	var newlen int
	toMap := !(f.ti.toArray || e.h.StructToArray)
	var mf map[string]interface{}
	if f.ti.mf {
		mf = rv2i(rv).(MissingFielder).CodecMissingFields()
		toMap = true
		newlen += len(mf)
	} else if f.ti.mfp {
		if rv.CanAddr() {
			mf = rv2i(rv.Addr()).(MissingFielder).CodecMissingFields()
		} else {
			// make a new addressable value of same one, and use it
			rv2 := reflect.New(rv.Type())
			rv2.Elem().Set(rv)
			mf = rv2i(rv2).(MissingFielder).CodecMissingFields()
		}
		toMap = true
		newlen += len(mf)
	}
	newlen += len(f.ti.sfiSrc)

	// Use sync.Pool to reduce allocating slices unnecessarily.
	// The cost of sync.Pool is less than the cost of new allocation.
	//
	// Each element of the array pools one of encStructPool(8|16|32|64).
	// It allows the re-use of slices up to 64 in length.
	// A performance cost of encoding structs was collecting
	// which values were empty and should be omitted.
	// We needed slices of reflect.Value and string to collect them.
	// This shared pool reduces the amount of unnecessary creation we do.
	// The cost is that of locking sometimes, but sync.Pool is efficient
	// enough to reduce thread contention.

	// fmt.Printf(">>>>>>>>>>>>>> encode.kStruct: newlen: %d\n", newlen)
	var spool sfiRvPooler
	var fkvs = spool.get(newlen)

	recur := e.h.RecursiveEmptyCheck
	sfn := structFieldNode{v: rv, update: false}

	var kv sfiRv
	var j int
	if toMap {
		newlen = 0
		for _, si := range f.ti.sfiSort { // use sorted array
			// kv.r = si.field(rv, false)
			kv.r = sfn.field(si)
			if si.omitEmpty() && isEmptyValue(kv.r, e.h.TypeInfos, recur, recur) {
				continue
			}
			kv.v = si // si.encName
			fkvs[newlen] = kv
			newlen++
		}
		var mflen int
		for k, v := range mf {
			if k == "" {
				delete(mf, k)
				continue
			}
			if f.ti.infoFieldOmitempty && isEmptyValue(reflect.ValueOf(v), e.h.TypeInfos, recur, recur) {
				delete(mf, k)
				continue
			}
			mflen++
		}
		// encode it all
		e.mapStart(newlen + mflen)
		for j = 0; j < newlen; j++ {
			kv = fkvs[j]
			e.mapElemKey()
			e.kStructFieldKey(f.ti.keyType, kv.v.encNameAsciiAlphaNum, kv.v.encName)
			e.mapElemValue()
			e.encodeValue(kv.r, nil)
		}
		// now, add the others
		for k, v := range mf {
			e.mapElemKey()
			e.kStructFieldKey(f.ti.keyType, false, k)
			e.mapElemValue()
			e.encode(v)
		}
		e.mapEnd()
	} else {
		newlen = len(f.ti.sfiSrc)
		// kv.v = nil
		for i, si := range f.ti.sfiSrc { // use unsorted array (to match sequence in struct)
			// kv.r = si.field(rv, false)
			kv.r = sfn.field(si)
			// use the zero value.
			// if a reference or struct, set to nil (so you do not output too much)
			if si.omitEmpty() && isEmptyValue(kv.r, e.h.TypeInfos, recur, recur) {
				switch kv.r.Kind() {
				case reflect.Struct, reflect.Interface, reflect.Ptr, reflect.Array, reflect.Map, reflect.Slice:
					kv.r = reflect.Value{} //encode as nil
				}
			}
			fkvs[i] = kv
		}
		// encode it all
		e.arrayStart(newlen)
		for j = 0; j < newlen; j++ {
			e.arrayElem()
			e.encodeValue(fkvs[j].r, nil)
		}
		e.arrayEnd()
	}

	// do not use defer. Instead, use explicit pool return at end of function.
	// defer has a cost we are trying to avoid.
	// If there is a panic and these slices are not returned, it is ok.
	spool.end()
}

func (e *Encoder) kMap(f *codecFnInfo, rv reflect.Value) {
	if rvisnil(rv) {
		e.e.EncodeNil()
		return
	}

	l := rv.Len()
	e.mapStart(l)
	if l == 0 {
		e.mapEnd()
		return
	}
	// var asSymbols bool
	// determine the underlying key and val encFn's for the map.
	// This eliminates some work which is done for each loop iteration i.e.
	// rv.Type(), ref.ValueOf(rt).Pointer(), then check map/list for fn.
	//
	// However, if kind is reflect.Interface, do not pre-determine the
	// encoding type, because preEncodeValue may break it down to
	// a concrete type and kInterface will bomb.
	var keyFn, valFn *codecFn
	rtval := f.ti.elem
	// rtkeyid := rt2id(f.ti.key)
	for rtval.Kind() == reflect.Ptr {
		rtval = rtval.Elem()
	}
	if rtval.Kind() != reflect.Interface {
		valFn = e.h.fn(rtval)
	}

	if e.h.Canonical {
		e.kMapCanonical(f.ti.key, f.ti.elem, rv, valFn)
		e.mapEnd()
		return
	}

	rtkey := f.ti.key
	var keyTypeIsString = stringTypId == rt2id(rtkey) // rtkeyid
	if !keyTypeIsString {
		for rtkey.Kind() == reflect.Ptr {
			rtkey = rtkey.Elem()
		}
		if rtkey.Kind() != reflect.Interface {
			// rtkeyid = rt2id(rtkey)
			keyFn = e.h.fn(rtkey)
		}
	}

	var rvk = mapAddressableRV(f.ti.key)  //, f.ti.key.Kind())
	var rvv = mapAddressableRV(f.ti.elem) //, f.ti.elem.Kind())

	it := mapRange(rv, rvk, rvv, true)

	for it.Next() {
		e.mapElemKey()
		if keyTypeIsString {
			if e.h.StringToRaw {
				e.e.EncodeStringBytesRaw(bytesView(it.Key().String()))
			} else {
				e.e.EncodeStringEnc(cUTF8, it.Key().String())
			}
		} else {
			e.encodeValue(it.Key(), keyFn) //
		}
		e.mapElemValue()
		iv := it.Value()
		e.encodeValue(iv, valFn)
	}

	e.mapEnd()
}

func (e *Encoder) kMapCanonical(rtkey, rtval reflect.Type, rv reflect.Value, valFn *codecFn) {
	// we previously did out-of-band if an extension was registered.
	// This is not necessary, as the natural kind is sufficient for ordering.

	rvv := mapAddressableRV(rtval)

	mks := rv.MapKeys()
	switch rtkey.Kind() {
	case reflect.Bool:
		mksv := make([]boolRv, len(mks))
		for i, k := range mks {
			v := &mksv[i]
			v.r = k
			v.v = k.Bool()
		}
		sort.Sort(boolRvSlice(mksv))
		for i := range mksv {
			e.mapElemKey()
			e.e.EncodeBool(mksv[i].v)
			e.mapElemValue()
			e.encodeValue(mapGet(rv, mksv[i].r, rvv), valFn) // e.encodeValue(rv.MapIndex(mksv[i].r), valFn)
		}
	case reflect.String:
		mksv := make([]stringRv, len(mks))
		for i, k := range mks {
			v := &mksv[i]
			v.r = k
			v.v = k.String()
		}
		sort.Sort(stringRvSlice(mksv))
		for i := range mksv {
			e.mapElemKey()
			if e.h.StringToRaw {
				e.e.EncodeStringBytesRaw(bytesView(mksv[i].v))
			} else {
				e.e.EncodeStringEnc(cUTF8, mksv[i].v)
			}
			e.mapElemValue()
			e.encodeValue(mapGet(rv, mksv[i].r, rvv), valFn) // e.encodeValue(rv.MapIndex(mksv[i].r), valFn)
		}
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint, reflect.Uintptr:
		mksv := make([]uint64Rv, len(mks))
		for i, k := range mks {
			v := &mksv[i]
			v.r = k
			v.v = k.Uint()
		}
		sort.Sort(uint64RvSlice(mksv))
		for i := range mksv {
			e.mapElemKey()
			e.e.EncodeUint(mksv[i].v)
			e.mapElemValue()
			e.encodeValue(mapGet(rv, mksv[i].r, rvv), valFn) // e.encodeValue(rv.MapIndex(mksv[i].r), valFn)
		}
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		mksv := make([]int64Rv, len(mks))
		for i, k := range mks {
			v := &mksv[i]
			v.r = k
			v.v = k.Int()
		}
		sort.Sort(int64RvSlice(mksv))
		for i := range mksv {
			e.mapElemKey()
			e.e.EncodeInt(mksv[i].v)
			e.mapElemValue()
			e.encodeValue(mapGet(rv, mksv[i].r, rvv), valFn) // e.encodeValue(rv.MapIndex(mksv[i].r), valFn)
		}
	case reflect.Float32:
		mksv := make([]float64Rv, len(mks))
		for i, k := range mks {
			v := &mksv[i]
			v.r = k
			v.v = k.Float()
		}
		sort.Sort(float64RvSlice(mksv))
		for i := range mksv {
			e.mapElemKey()
			e.e.EncodeFloat32(float32(mksv[i].v))
			e.mapElemValue()
			e.encodeValue(mapGet(rv, mksv[i].r, rvv), valFn) // e.encodeValue(rv.MapIndex(mksv[i].r), valFn)
		}
	case reflect.Float64:
		mksv := make([]float64Rv, len(mks))
		for i, k := range mks {
			v := &mksv[i]
			v.r = k
			v.v = k.Float()
		}
		sort.Sort(float64RvSlice(mksv))
		for i := range mksv {
			e.mapElemKey()
			e.e.EncodeFloat64(mksv[i].v)
			e.mapElemValue()
			e.encodeValue(mapGet(rv, mksv[i].r, rvv), valFn) // e.encodeValue(rv.MapIndex(mksv[i].r), valFn)
		}
	case reflect.Struct:
		if rv.Type() == timeTyp {
			mksv := make([]timeRv, len(mks))
			for i, k := range mks {
				v := &mksv[i]
				v.r = k
				v.v = rv2i(k).(time.Time)
			}
			sort.Sort(timeRvSlice(mksv))
			for i := range mksv {
				e.mapElemKey()
				e.e.EncodeTime(mksv[i].v)
				e.mapElemValue()
				e.encodeValue(mapGet(rv, mksv[i].r, rvv), valFn)
			}
			break
		}
		fallthrough
	default:
		// out-of-band
		// first encode each key to a []byte first, then sort them, then record
		var bufp bytesBufPooler
		var mksv []byte = bufp.get(len(mks) * 16)[:0]
		e2 := NewEncoderBytes(&mksv, e.hh)
		mksbv := make([]bytesRv, len(mks))
		for i, k := range mks {
			v := &mksbv[i]
			l := len(mksv)
			e2.MustEncode(k)
			v.r = k
			v.v = mksv[l:]
		}
		sort.Sort(bytesRvSlice(mksbv))
		for j := range mksbv {
			e.mapElemKey()
			e.asis(mksbv[j].v)
			e.mapElemValue()
			e.encodeValue(mapGet(rv, mksbv[j].r, rvv), valFn)
		}
		bufp.end()
	}
}

// Encoder writes an object to an output stream in a supported format.
//
// Encoder is NOT safe for concurrent use i.e. a Encoder cannot be used
// concurrently in multiple goroutines.
//
// However, as Encoder could be allocation heavy to initialize, a Reset method is provided
// so its state can be reused to decode new input streams repeatedly.
// This is the idiomatic way to use.
type Encoder struct {
	panicHdl
	// hopefully, reduce derefencing cost by laying the encWriter inside the Encoder
	e encDriver

	// NOTE: Encoder shouldn't call it's write methods,
	// as the handler MAY need to do some coordination.
	// w *encWriterSwitch

	// bw *bufio.Writer
	as encDriverAsis

	jenc *jsonEncDriver
	h    *BasicHandle
	hh   Handle

	// ---- cpu cache line boundary
	encWriterSwitch

	err error

	// ---- cpu cache line boundary
	// ---- writable fields during execution --- *try* to keep in sep cache line
	ci set // holds set of addresses found during an encoding (if CheckCircularRef=true)
	// cidef [1]interface{} // default ci

	b [(5 * 8)]byte // for encoding chan byte, (non-addressable) [N]byte, etc

	// ---- cpu cache line boundary?
	// b [scratchByteArrayLen]byte
	// _ [cacheLineSize - scratchByteArrayLen]byte // padding
	// b [cacheLineSize - (8 * 0)]byte // used for encoding a chan or (non-addressable) array of bytes
}

// NewEncoder returns an Encoder for encoding into an io.Writer.
//
// For efficiency, Users are encouraged to configure WriterBufferSize on the handle
// OR pass in a memory buffered writer (eg bufio.Writer, bytes.Buffer).
func NewEncoder(w io.Writer, h Handle) *Encoder {
	e := newEncoder(h)
	e.Reset(w)
	return e
}

// NewEncoderBytes returns an encoder for encoding directly and efficiently
// into a byte slice, using zero-copying to temporary slices.
//
// It will potentially replace the output byte slice pointed to.
// After encoding, the out parameter contains the encoded contents.
func NewEncoderBytes(out *[]byte, h Handle) *Encoder {
	e := newEncoder(h)
	e.ResetBytes(out)
	return e
}

func newEncoder(h Handle) *Encoder {
	e := &Encoder{h: basicHandle(h), err: errEncoderNotInitialized}
	e.bytes = true
	if useFinalizers {
		runtime.SetFinalizer(e, (*Encoder).finalize)
	}
	// e.w = &e.encWriterSwitch
	e.hh = h
	e.esep = h.hasElemSeparators()

	return e
}

func (e *Encoder) w() *encWriterSwitch {
	return &e.encWriterSwitch
}

func (e *Encoder) resetCommon() {
	// e.w = &e.encWriterSwitch
	if e.e == nil || e.hh.recreateEncDriver(e.e) {
		e.e = e.hh.newEncDriver(e)
		e.as, e.isas = e.e.(encDriverAsis)
		// e.cr, _ = e.e.(containerStateRecv)
	}

	if e.ci == nil {
		// e.ci = (set)(e.cidef[:0])
	} else {
		e.ci = e.ci[:0]
	}

	e.be = e.hh.isBinary()
	e.jenc = nil
	_, e.js = e.hh.(*JsonHandle)
	if e.js {
		e.jenc = e.e.(interface{ getJsonEncDriver() *jsonEncDriver }).getJsonEncDriver()
	}
	e.e.reset()
	e.c = 0
	e.err = nil
}

// Reset resets the Encoder with a new output stream.
//
// This accommodates using the state of the Encoder,
// where it has "cached" information about sub-engines.
func (e *Encoder) Reset(w io.Writer) {
	if w == nil {
		return
	}
	// var ok bool
	e.bytes = false
	if e.wf == nil {
		e.wf = new(bufioEncWriter)
	}
	// e.typ = entryTypeUnset
	// if e.h.WriterBufferSize > 0 {
	// 	// bw := bufio.NewWriterSize(w, e.h.WriterBufferSize)
	// 	// e.wi.bw = bw
	// 	// e.wi.sw = bw
	// 	// e.wi.fw = bw
	// 	// e.wi.ww = bw
	// 	if e.wf == nil {
	// 		e.wf = new(bufioEncWriter)
	// 	}
	// 	e.wf.reset(w, e.h.WriterBufferSize)
	// 	e.typ = entryTypeBufio
	// } else {
	// 	if e.wi == nil {
	// 		e.wi = new(ioEncWriter)
	// 	}
	// 	e.wi.reset(w)
	// 	e.typ = entryTypeIo
	// }
	e.wf.reset(w, e.h.WriterBufferSize)
	// e.typ = entryTypeBufio

	// e.w = e.wi
	e.resetCommon()
}

// ResetBytes resets the Encoder with a new destination output []byte.
func (e *Encoder) ResetBytes(out *[]byte) {
	if out == nil {
		return
	}
	var in []byte = *out
	if in == nil {
		in = make([]byte, defEncByteBufSize)
	}
	e.bytes = true
	// e.typ = entryTypeBytes
	e.wb.reset(in, out)
	// e.w = &e.wb
	e.resetCommon()
}

// Encode writes an object into a stream.
//
// Encoding can be configured via the struct tag for the fields.
// The key (in the struct tags) that we look at is configurable.
//
// By default, we look up the "codec" key in the struct field's tags,
// and fall bak to the "json" key if "codec" is absent.
// That key in struct field's tag value is the key name,
// followed by an optional comma and options.
//
// To set an option on all fields (e.g. omitempty on all fields), you
// can create a field called _struct, and set flags on it. The options
// which can be set on _struct are:
//    - omitempty: so all fields are omitted if empty
//    - toarray: so struct is encoded as an array
//    - int: so struct key names are encoded as signed integers (instead of strings)
//    - uint: so struct key names are encoded as unsigned integers (instead of strings)
//    - float: so struct key names are encoded as floats (instead of strings)
// More details on these below.
//
// Struct values "usually" encode as maps. Each exported struct field is encoded unless:
//    - the field's tag is "-", OR
//    - the field is empty (empty or the zero value) and its tag specifies the "omitempty" option.
//
// When encoding as a map, the first string in the tag (before the comma)
// is the map key string to use when encoding.
// ...
// This key is typically encoded as a string.
// However, there are instances where the encoded stream has mapping keys encoded as numbers.
// For example, some cbor streams have keys as integer codes in the stream, but they should map
// to fields in a structured object. Consequently, a struct is the natural representation in code.
// For these, configure the struct to encode/decode the keys as numbers (instead of string).
// This is done with the int,uint or float option on the _struct field (see above).
//
// However, struct values may encode as arrays. This happens when:
//    - StructToArray Encode option is set, OR
//    - the tag on the _struct field sets the "toarray" option
// Note that omitempty is ignored when encoding struct values as arrays,
// as an entry must be encoded for each field, to maintain its position.
//
// Values with types that implement MapBySlice are encoded as stream maps.
//
// The empty values (for omitempty option) are false, 0, any nil pointer
// or interface value, and any array, slice, map, or string of length zero.
//
// Anonymous fields are encoded inline except:
//    - the struct tag specifies a replacement name (first value)
//    - the field is of an interface type
//
// Examples:
//
//      // NOTE: 'json:' can be used as struct tag key, in place 'codec:' below.
//      type MyStruct struct {
//          _struct bool    `codec:",omitempty"`   //set omitempty for every field
//          Field1 string   `codec:"-"`            //skip this field
//          Field2 int      `codec:"myName"`       //Use key "myName" in encode stream
//          Field3 int32    `codec:",omitempty"`   //use key "Field3". Omit if empty.
//          Field4 bool     `codec:"f4,omitempty"` //use key "f4". Omit if empty.
//          io.Reader                              //use key "Reader".
//          MyStruct        `codec:"my1"           //use key "my1".
//          MyStruct                               //inline it
//          ...
//      }
//
//      type MyStruct struct {
//          _struct bool    `codec:",toarray"`     //encode struct as an array
//      }
//
//      type MyStruct struct {
//          _struct bool    `codec:",uint"`        //encode struct with "unsigned integer" keys
//          Field1 string   `codec:"1"`            //encode Field1 key using: EncodeInt(1)
//          Field2 string   `codec:"2"`            //encode Field2 key using: EncodeInt(2)
//      }
//
// The mode of encoding is based on the type of the value. When a value is seen:
//   - If a Selfer, call its CodecEncodeSelf method
//   - If an extension is registered for it, call that extension function
//   - If implements encoding.(Binary|Text|JSON)Marshaler, call Marshal(Binary|Text|JSON) method
//   - Else encode it based on its reflect.Kind
//
// Note that struct field names and keys in map[string]XXX will be treated as symbols.
// Some formats support symbols (e.g. binc) and will properly encode the string
// only once in the stream, and use a tag to refer to it thereafter.
func (e *Encoder) Encode(v interface{}) (err error) {
	// tried to use closure, as runtime optimizes defer with no params.
	// This seemed to be causing weird issues (like circular reference found, unexpected panic, etc).
	// Also, see https://github.com/golang/go/issues/14939#issuecomment-417836139
	// defer func() { e.deferred(&err) }() }
	// { x, y := e, &err; defer func() { x.deferred(y) }() }
	if e.err != nil {
		return e.err
	}
	if recoverPanicToErr {
		defer func() {
			// if error occurred during encoding, return that error;
			// else if error occurred on end'ing (i.e. during flush), return that error.
			err = e.w().endErr()
			x := recover()
			if x == nil {
				if e.err != err {
					e.err = err
				}
			} else {
				panicValToErr(e, x, &e.err)
				if e.err != err {
					err = e.err
				}
			}
		}()
	}

	// defer e.deferred(&err)
	e.mustEncode(v)
	return
}

// MustEncode is like Encode, but panics if unable to Encode.
// This provides insight to the code location that triggered the error.
func (e *Encoder) MustEncode(v interface{}) {
	if e.err != nil {
		panic(e.err)
	}
	e.mustEncode(v)
}

func (e *Encoder) mustEncode(v interface{}) {
	if e.wf == nil {
		e.encode(v)
		e.e.atEndOfEncode()
		e.w().end()
		return
	}

	if e.wf.buf == nil {
		e.wf.buf = e.wf.bytesBufPooler.get(e.wf.sz)
	}
	e.wf.calls++
	e.encode(v)
	e.wf.calls--
	if e.wf.calls == 0 {
		e.e.atEndOfEncode()
		e.w().end()
		if !e.h.ExplicitRelease {
			e.wf.release()
		}
	}
}

// func (e *Encoder) deferred(err1 *error) {
// 	e.w().end()
// 	if recoverPanicToErr {
// 		if x := recover(); x != nil {
// 			panicValToErr(e, x, err1)
// 			panicValToErr(e, x, &e.err)
// 		}
// 	}
// }

//go:noinline -- as it is run by finalizer
func (e *Encoder) finalize() {
	e.Release()
}

// Release releases shared (pooled) resources.
//
// It is important to call Release() when done with an Encoder, so those resources
// are released instantly for use by subsequently created Encoders.
func (e *Encoder) Release() {
	if e.wf != nil {
		e.wf.release()
	}
}

func (e *Encoder) encode(iv interface{}) {
	// a switch with only concrete types can be optimized.
	// consequently, we deal with nil and interfaces outside the switch.

	if iv == nil || definitelyNil(iv) {
		e.e.EncodeNil()
		return
	}

	switch v := iv.(type) {
	// case nil:
	// case Selfer:
	case Raw:
		e.rawBytes(v)
	case reflect.Value:
		e.encodeValue(v, nil)

	case string:
		if e.h.StringToRaw {
			e.e.EncodeStringBytesRaw(bytesView(v))
		} else {
			e.e.EncodeStringEnc(cUTF8, v)
		}
	case bool:
		e.e.EncodeBool(v)
	case int:
		e.e.EncodeInt(int64(v))
	case int8:
		e.e.EncodeInt(int64(v))
	case int16:
		e.e.EncodeInt(int64(v))
	case int32:
		e.e.EncodeInt(int64(v))
	case int64:
		e.e.EncodeInt(v)
	case uint:
		e.e.EncodeUint(uint64(v))
	case uint8:
		e.e.EncodeUint(uint64(v))
	case uint16:
		e.e.EncodeUint(uint64(v))
	case uint32:
		e.e.EncodeUint(uint64(v))
	case uint64:
		e.e.EncodeUint(v)
	case uintptr:
		e.e.EncodeUint(uint64(v))
	case float32:
		e.e.EncodeFloat32(v)
	case float64:
		e.e.EncodeFloat64(v)
	case time.Time:
		e.e.EncodeTime(v)
	case []uint8:
		e.e.EncodeStringBytesRaw(v)

	case *Raw:
		e.rawBytes(*v)

	case *string:
		if e.h.StringToRaw {
			e.e.EncodeStringBytesRaw(bytesView(*v))
		} else {
			e.e.EncodeStringEnc(cUTF8, *v)
		}
	case *bool:
		e.e.EncodeBool(*v)
	case *int:
		e.e.EncodeInt(int64(*v))
	case *int8:
		e.e.EncodeInt(int64(*v))
	case *int16:
		e.e.EncodeInt(int64(*v))
	case *int32:
		e.e.EncodeInt(int64(*v))
	case *int64:
		e.e.EncodeInt(*v)
	case *uint:
		e.e.EncodeUint(uint64(*v))
	case *uint8:
		e.e.EncodeUint(uint64(*v))
	case *uint16:
		e.e.EncodeUint(uint64(*v))
	case *uint32:
		e.e.EncodeUint(uint64(*v))
	case *uint64:
		e.e.EncodeUint(*v)
	case *uintptr:
		e.e.EncodeUint(uint64(*v))
	case *float32:
		e.e.EncodeFloat32(*v)
	case *float64:
		e.e.EncodeFloat64(*v)
	case *time.Time:
		e.e.EncodeTime(*v)

	case *[]uint8:
		e.e.EncodeStringBytesRaw(*v)

	default:
		if v, ok := iv.(Selfer); ok {
			v.CodecEncodeSelf(e)
		} else if !fastpathEncodeTypeSwitch(iv, e) {
			// checkfastpath=true (not false), as underlying slice/map type may be fast-path
			e.encodeValue(reflect.ValueOf(iv), nil)
		}
	}
}

func (e *Encoder) encodeValue(rv reflect.Value, fn *codecFn) {
	// if a valid fn is passed, it MUST BE for the dereferenced type of rv

	// We considered using a uintptr (a pointer) retrievable via rv.UnsafeAddr.
	// However, it is possible for the same pointer to point to 2 different types e.g.
	//    type T struct { tHelper }
	//    Here, for var v T; &v and &v.tHelper are the same pointer.
	// Consequently, we need a tuple of type and pointer, which interface{} natively provides.
	var sptr interface{} // uintptr
	var rvp reflect.Value
	var rvpValid bool
TOP:
	switch rv.Kind() {
	case reflect.Ptr:
		if rvisnil(rv) {
			e.e.EncodeNil()
			return
		}
		rvpValid = true
		rvp = rv
		rv = rv.Elem()
		if e.h.CheckCircularRef && rv.Kind() == reflect.Struct {
			sptr = rv2i(rvp) // rv.UnsafeAddr()
			break TOP
		}
		goto TOP
	case reflect.Interface:
		if rvisnil(rv) {
			e.e.EncodeNil()
			return
		}
		rv = rv.Elem()
		goto TOP
	case reflect.Slice, reflect.Map:
		if rvisnil(rv) {
			e.e.EncodeNil()
			return
		}
	case reflect.Invalid, reflect.Func:
		e.e.EncodeNil()
		return
	}

	if sptr != nil && (&e.ci).add(sptr) {
		// e.errorf("circular reference found: # %d", sptr)
		e.errorf("circular reference found: # %p, %T", sptr, sptr)
	}

	if fn == nil {
		rt := rv.Type()
		fn = e.h.fn(rt)
	}
	if fn.i.addrE {
		if rvpValid {
			fn.fe(e, &fn.i, rvp)
		} else if rv.CanAddr() {
			fn.fe(e, &fn.i, rv.Addr())
		} else {
			rv2 := reflect.New(rv.Type())
			rv2.Elem().Set(rv)
			fn.fe(e, &fn.i, rv2)
		}
	} else {
		fn.fe(e, &fn.i, rv)
	}
	if sptr != 0 {
		(&e.ci).remove(sptr)
	}
}

// func (e *Encoder) marshal(bs []byte, fnerr error, asis bool, c charEncoding) {
// 	if fnerr != nil {
// 		panic(fnerr)
// 	}
// 	if bs == nil {
// 		e.e.EncodeNil()
// 	} else if asis {
// 		e.asis(bs)
// 	} else {
// 		e.e.EncodeStringBytesRaw(bs)
// 	}
// }

func (e *Encoder) marshalUtf8(bs []byte, fnerr error) {
	if fnerr != nil {
		panic(fnerr)
	}
	if bs == nil {
		e.e.EncodeNil()
	} else {
		e.e.EncodeStringEnc(cUTF8, stringView(bs))
	}
}

func (e *Encoder) marshalAsis(bs []byte, fnerr error) {
	if fnerr != nil {
		panic(fnerr)
	}
	if bs == nil {
		e.e.EncodeNil()
	} else {
		e.asis(bs)
	}
}

func (e *Encoder) marshalRaw(bs []byte, fnerr error) {
	if fnerr != nil {
		panic(fnerr)
	}
	if bs == nil {
		e.e.EncodeNil()
	} else {
		e.e.EncodeStringBytesRaw(bs)
	}
}

func (e *Encoder) asis(v []byte) {
	if e.isas {
		e.as.EncodeAsis(v)
	} else {
		e.w().writeb(v)
	}
}

func (e *Encoder) rawBytes(vv Raw) {
	v := []byte(vv)
	if !e.h.Raw {
		e.errorf("Raw values cannot be encoded: %v", v)
	}
	e.asis(v)
}

func (e *Encoder) wrapErr(v interface{}, err *error) {
	*err = encodeError{codecError{name: e.hh.Name(), err: v}}
}

// ---- container tracker methods
// Note: We update the .c after calling the callback.
// This way, the callback can know what the last status was.

func (e *Encoder) mapStart(length int) {
	e.e.WriteMapStart(length)
	e.c = containerMapStart
}

func (e *Encoder) mapElemKey() {
	if e.js {
		e.jenc.WriteMapElemKey()
	}
	e.c = containerMapKey
}

func (e *Encoder) mapElemValue() {
	if e.js {
		e.jenc.WriteMapElemValue()
	}
	e.c = containerMapValue
}

// // Note: This is harder to inline, as there are 2 function calls inside.
// func (e *Encoder) mapElemKeyOrValue(j uint8) {
// 	if j == 0 {
// 		if e.js {
// 			e.jenc.WriteMapElemKey()
// 		}
// 		e.c = containerMapKey
// 	} else {
// 		if e.js {
// 			e.jenc.WriteMapElemValue()
// 		}
// 		e.c = containerMapValue
// 	}
// }

func (e *Encoder) mapEnd() {
	e.e.WriteMapEnd()
	e.c = containerMapEnd
	e.c = 0
}

func (e *Encoder) arrayStart(length int) {
	e.e.WriteArrayStart(length)
	e.c = containerArrayStart
}

func (e *Encoder) arrayElem() {
	if e.js {
		e.jenc.WriteArrayElem()
	}
	e.c = containerArrayElem
}

func (e *Encoder) arrayEnd() {
	e.e.WriteArrayEnd()
	e.c = 0
	e.c = containerArrayEnd
}

// ----------

func (e *Encoder) sideEncode(v interface{}, bs *[]byte) {
	rv := baseRV(v)
	e2 := NewEncoderBytes(bs, e.hh)
	e2.encodeValue(rv, e.h.fnNoExt(rv.Type()))
	e2.e.atEndOfEncode()
	e2.w().end()
}

func encStructFieldKey(encName string, ee encDriver, w *encWriterSwitch,
	keyType valueType, encNameAsciiAlphaNum bool, js bool) {
	var m must
	// use if-else-if, not switch (which compiles to binary-search)
	// since keyType is typically valueTypeString, branch prediction is pretty good.
	if keyType == valueTypeString {
		if js && encNameAsciiAlphaNum { // keyType == valueTypeString
			w.writeqstr(encName)
			// ----
			// w.writen1('"')
			// w.writestr(encName)
			// w.writen1('"')
			// ----
			// w.writestr(`"` + encName + `"`)
			// ----
			// // do concat myself, so it is faster than the generic string concat
			// b := make([]byte, len(encName)+2)
			// copy(b[1:], encName)
			// b[0] = '"'
			// b[len(b)-1] = '"'
			// w.writeb(b)
		} else { // keyType == valueTypeString
			ee.EncodeStringEnc(cUTF8, encName)
		}
	} else if keyType == valueTypeInt {
		ee.EncodeInt(m.Int(strconv.ParseInt(encName, 10, 64)))
	} else if keyType == valueTypeUint {
		ee.EncodeUint(m.Uint(strconv.ParseUint(encName, 10, 64)))
	} else if keyType == valueTypeFloat {
		ee.EncodeFloat64(m.Float(strconv.ParseFloat(encName, 64)))
	}
}

// type encExtPreambler interface {
// 	encodeExtPreamble(tag uint8, length int)
// }

// func encBytesExt(rv interface{}, xtag uint64, ext Ext, h Handle, e encDriver) {
// 	var bs []byte
// 	var bufp bytesBufPooler
// 	if ext == SelfExt {
// 		bs = bufp.get(1024)[:0]
// 		rv2 := reflect.ValueOf(v)
// 		NewEncoderBytes(&bs, h).encodeValue(rv2, h.fnNoExt(rv2.Type()))
// 	} else {
// 		bs = ext.WriteExt(v)
// 	}
// 	if bs == nil {
// 		e.EncodeNil()
// 		return
// 	}
// 	if e.h.WriteExt {
// 		e.encodeExtPreamble(uint8(xtag), len(bs))
// 		e.w.writeb(bs)
// 	} else {
// 		e.EncodeStringBytesRaw(bs)
// 	}
// 	if ext == SelfExt {
// 		bufp.end()
// 	}
// }

// func encStringAsRawBytesMaybe(ee encDriver, s string, stringToRaw bool) {
// 	if stringToRaw {
// 		ee.EncodeStringBytesRaw(bytesView(s))
// 	} else {
// 		ee.EncodeStringEnc(cUTF8, s)
// 	}
// }