/*
Copyright 2016 Stanislav Liberman

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package atomic

import (
	"fmt"
	"github.com/lirm/aeron-go/aeron/util"
	"log"
	"reflect"
	"sync/atomic"
	"unsafe"
)

type Buffer struct {
	bufferPtr unsafe.Pointer
	length    int32
}

/*
	Options for calling
		MakeAtomicBuffer(Pointer)
		MakeAtomicBuffer([]byte)
		MakeAtomicBuffer(Pointer, len)
		MakeAtomicBuffer([]byte, len)
*/
func MakeBuffer(args ...interface{}) *Buffer {
	var bufPtr unsafe.Pointer
	var bufLen int32

	switch len(args) {
	case 1:
		// just wrap
		switch reflect.TypeOf(args[0]) {
		case reflect.TypeOf(unsafe.Pointer(nil)):
			bufPtr = unsafe.Pointer(args[0].(unsafe.Pointer))

		case reflect.TypeOf(([]uint8)(nil)):
			arr := ([]byte)(args[0].([]uint8))
			bufPtr = unsafe.Pointer(&arr[0])
			bufLen = int32(len(arr))
		}
	case 2:
		// wrap with length
		if reflect.TypeOf(args[1]).ConvertibleTo(reflect.TypeOf(int32(0))) {
			v := reflect.ValueOf(args[1])
			t := reflect.TypeOf(int32(0))
			bufLen = int32(v.Convert(t).Int())
		}
		switch reflect.TypeOf(args[0]) {
		case reflect.TypeOf(unsafe.Pointer(nil)):
			bufPtr = unsafe.Pointer(args[0].(unsafe.Pointer))

		case reflect.TypeOf(([]uint8)(nil)):
			arr := ([]byte)(args[0].([]uint8))
			bufPtr = unsafe.Pointer(&arr[0])
		}
	case 3:
		// wrap with offset and length
	}

	buf := new(Buffer)
	return buf.Wrap(bufPtr, bufLen)
}

func (buf *Buffer) Wrap(buffer unsafe.Pointer, length int32) *Buffer {
	buf.bufferPtr = buffer
	buf.length = length
	return buf
}

func (buf *Buffer) Ptr() unsafe.Pointer {
	return buf.bufferPtr
}

func (buf *Buffer) Capacity() int32 {
	return buf.length
}

func (buf *Buffer) Fill(b uint8) {
	if buf.length == 0 {
		return
	}
	for ix := 0; ix < int(buf.length); ix++ {
		uptr := unsafe.Pointer(uintptr(buf.bufferPtr) + uintptr(ix))
		*(*uint8)(uptr) = b
	}
}

func (buf *Buffer) GetUInt8(offset int32) uint8 {
	buf.BoundsCheck(offset, 1)

	uptr := unsafe.Pointer(uintptr(buf.bufferPtr) + uintptr(offset))

	return *(*uint8)(uptr)
}

func (buf *Buffer) GetUInt16(offset int32) uint16 {
	buf.BoundsCheck(offset, 2)

	uptr := unsafe.Pointer(uintptr(buf.bufferPtr) + uintptr(offset))

	return *(*uint16)(uptr)
}

func (buf *Buffer) GetInt32(offset int32) int32 {
	buf.BoundsCheck(offset, 4)

	uptr := unsafe.Pointer(uintptr(buf.bufferPtr) + uintptr(offset))

	return *(*int32)(uptr)
}

func (buf *Buffer) GetInt64(offset int32) int64 {
	buf.BoundsCheck(offset, 8)

	uptr := unsafe.Pointer(uintptr(buf.bufferPtr) + uintptr(offset))

	return *(*int64)(uptr)
}

func (buf *Buffer) PutUInt8(offset int32, value uint8) {
	buf.BoundsCheck(offset, 1)

	uptr := unsafe.Pointer(uintptr(buf.bufferPtr) + uintptr(offset))

	*(*uint8)(uptr) = value
}

func (buf *Buffer) PutInt8(offset int32, value int8) {
	buf.BoundsCheck(offset, 1)

	uptr := unsafe.Pointer(uintptr(buf.bufferPtr) + uintptr(offset))

	*(*int8)(uptr) = value
}

func (buf *Buffer) PutUInt16(offset int32, value uint16) {
	buf.BoundsCheck(offset, 2)

	uptr := unsafe.Pointer(uintptr(buf.bufferPtr) + uintptr(offset))

	*(*uint16)(uptr) = value
}

func (buf *Buffer) PutInt32(offset int32, value int32) {
	buf.BoundsCheck(offset, 4)

	uptr := unsafe.Pointer(uintptr(buf.bufferPtr) + uintptr(offset))

	*(*int32)(uptr) = value
}

func (buf *Buffer) PutInt64(offset int32, value int64) {
	buf.BoundsCheck(offset, 8)

	uptr := unsafe.Pointer(uintptr(buf.bufferPtr) + uintptr(offset))

	*(*int64)(uptr) = value
}

func (buf *Buffer) GetAndAddInt64(offset int32, delta int64) int64 {
	buf.BoundsCheck(offset, 8)

	uptr := unsafe.Pointer(uintptr(buf.bufferPtr) + uintptr(offset))
	newVal := atomic.AddUint64((*uint64)(uptr), uint64(delta))

	return int64(newVal) - delta
}

func (buf *Buffer) GetInt32Volatile(offset int32) int32 {
	buf.BoundsCheck(offset, 4)

	uptr := unsafe.Pointer(uintptr(buf.bufferPtr) + uintptr(offset))
	cur := atomic.LoadUint32((*uint32)(uptr))

	return int32(cur)
}

func (buf *Buffer) GetInt64Volatile(offset int32) int64 {
	buf.BoundsCheck(offset, 8)

	uptr := unsafe.Pointer(uintptr(buf.bufferPtr) + uintptr(offset))
	cur := atomic.LoadUint64((*uint64)(uptr))

	return int64(cur)
}

func (buf *Buffer) PutInt64Ordered(offset int32, value int64) {
	buf.BoundsCheck(offset, 8)

	uptr := unsafe.Pointer(uintptr(buf.bufferPtr) + uintptr(offset))
	atomic.StoreInt64((*int64)(uptr), value)
}

func (buf *Buffer) PutInt32Ordered(offset int32, value int32) {
	buf.BoundsCheck(offset, 4)

	uptr := unsafe.Pointer(uintptr(buf.bufferPtr) + uintptr(offset))
	atomic.StoreInt32((*int32)(uptr), value)
}

func (buf *Buffer) PutIntOrdered(offset int32, value int) {
	buf.BoundsCheck(offset, 4)

	uptr := unsafe.Pointer(uintptr(buf.bufferPtr) + uintptr(offset))
	atomic.StoreInt32((*int32)(uptr), int32(value))
}

func (buf *Buffer) CompareAndSetInt64(offset int32, expectedValue, updateValue int64) bool {
	buf.BoundsCheck(offset, 8)

	uptr := unsafe.Pointer(uintptr(buf.bufferPtr) + uintptr(offset))
	return atomic.CompareAndSwapInt64((*int64)(uptr), expectedValue, updateValue)
}

func (buf *Buffer) CompareAndSetInt32(offset int32, expectedValue, updateValue int32) bool {
	buf.BoundsCheck(offset, 4)

	uptr := unsafe.Pointer(uintptr(buf.bufferPtr) + uintptr(offset))
	return atomic.CompareAndSwapInt32((*int32)(uptr), expectedValue, updateValue)
}

func (buf *Buffer) PutBytes(index int32, srcBuffer *Buffer, srcint32 int32, length int32) {
	buf.BoundsCheck(index, length)
	srcBuffer.BoundsCheck(srcint32, length)

	util.Memcpy(uintptr(buf.bufferPtr)+uintptr(index), uintptr(srcBuffer.bufferPtr)+uintptr(srcint32), length)
}

func (buf *Buffer) GetBytesArray(offset int32, length int32) []byte {
	buf.BoundsCheck(offset, length)

	bArr := make([]byte, length)
	for ix := 0; ix < int(length); ix++ {
		uptr := unsafe.Pointer(uintptr(buf.bufferPtr) + uintptr(offset) + uintptr(ix))
		bArr[ix] = *(*uint8)(uptr)
	}

	return bArr
}

func (buf *Buffer) PutBytesArray(index int32, arr *[]byte, srcint32 int32, length int32) {
	buf.BoundsCheck(index, length)
	boundsCheck(srcint32, length, int32(len(*arr)))

	bArr := *arr

	util.Memcpy(uintptr(buf.bufferPtr)+uintptr(index), uintptr(unsafe.Pointer(&bArr[0]))+uintptr(srcint32), length)
}

func (buf *Buffer) BoundsCheck(index int32, length int32) {
	if (index + length) > buf.length {
		log.Fatal(fmt.Sprintf("int32 Out of Bounds[%p]. int32: %d + %d Capacity: %d", buf, index, length, buf.length))
	}
}

func boundsCheck(index int32, length int32, myLength int32) {
	if (index + length) > myLength {
		log.Fatal(fmt.Sprintf("int32 Out of Bounds. int32: %d + %d Capacity: %d", index, length, myLength))
	}
}
