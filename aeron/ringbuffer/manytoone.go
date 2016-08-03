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

package rb

import (
	"fmt"
	"github.com/lirm/aeron-go/aeron/atomic"
	"github.com/lirm/aeron-go/aeron/util"
)

const InsufficientCapacity int32 = -2

var Descriptor = struct {
	tailPositionOffset       int32
	headCachePositionOffset  int32
	headPositionOffset       int32
	correlationCounterOffset int32
	consumerHeartbeatOffset  int32
	trailerLength            int32
}{
	util.CacheLineLength * 2,
	util.CacheLineLength * 4,
	util.CacheLineLength * 6,
	util.CacheLineLength * 8,
	util.CacheLineLength * 10,
	util.CacheLineLength * 12,
}

type ManyToOne struct {
	buffer                    *atomic.Buffer
	capacity                  int32
	maxMsgLength              int32
	headPositionIndex         int32
	headCachePositionIndex    int32
	tailPositionIndex         int32
	correlationIDCounterIndex int32
	consumerHeartbeatIndex    int32
}

func (buf *ManyToOne) Init(buffer *atomic.Buffer) *ManyToOne {

	buf.buffer = buffer
	buf.capacity = buffer.Capacity() - Descriptor.trailerLength

	util.IsPowerOfTwo(buf.capacity)

	buf.maxMsgLength = buf.capacity / 8
	buf.tailPositionIndex = buf.capacity + Descriptor.tailPositionOffset
	buf.headCachePositionIndex = buf.capacity + Descriptor.headCachePositionOffset
	buf.headPositionIndex = buf.capacity + Descriptor.headPositionOffset
	buf.correlationIDCounterIndex = buf.capacity + Descriptor.correlationCounterOffset
	buf.consumerHeartbeatIndex = buf.capacity + Descriptor.consumerHeartbeatOffset

	return buf
}

func (buf *ManyToOne) NextCorrelationID() int64 {
	return buf.buffer.GetAndAddInt64(buf.correlationIDCounterIndex, 1)
}

func (buf *ManyToOne) SetConsumerHeartbeatTime(time int64) {
	buf.buffer.PutInt64Ordered(buf.consumerHeartbeatIndex, time)
}

func (buf *ManyToOne) ConsumerHeartbeatTime() int64 {
	return buf.buffer.GetInt64Volatile(buf.consumerHeartbeatIndex)
}

func (buf *ManyToOne) ProducerPosition() int64 {
	return buf.buffer.GetInt64Volatile(buf.tailPositionIndex)
}

func (buf *ManyToOne) ConsumerPosition() int64 {
	return buf.buffer.GetInt64Volatile(buf.headPositionIndex)
}

func (buf *ManyToOne) Capacity() int32 {
	return buf.capacity
}

func (buf *ManyToOne) claimCapacity(requiredCapacity int32) int32 {

	mask := buf.capacity - 1
	head := buf.buffer.GetInt64Volatile(buf.headCachePositionIndex)

	var tail int64
	var tailIndex int32
	var padding int32

	for ok := true; ok; ok = !buf.buffer.CompareAndSetInt64(buf.tailPositionIndex, tail, tail+int64(requiredCapacity)+int64(padding)) {
		tail = buf.buffer.GetInt64Volatile(buf.tailPositionIndex)
		availableCapacity := buf.capacity - int32(tail-head)

		if requiredCapacity > availableCapacity {
			head = buf.buffer.GetInt64Volatile(buf.headPositionIndex)

			if requiredCapacity > (buf.capacity - int32(tail-head)) {
				return InsufficientCapacity
			}

			buf.buffer.PutInt64Ordered(buf.headCachePositionIndex, head)
		}

		padding = 0
		tailIndex = int32(tail & int64(mask))
		toBufferEndLength := buf.capacity - tailIndex

		if requiredCapacity > toBufferEndLength {
			headIndex := int32(head & int64(mask))

			if requiredCapacity > headIndex {
				head = buf.buffer.GetInt64Volatile(buf.headPositionIndex)
				headIndex = int32(head & int64(mask))

				if requiredCapacity > headIndex {
					return InsufficientCapacity
				}

				buf.buffer.PutInt64Ordered(buf.headCachePositionIndex, head)
			}

			padding = toBufferEndLength
		}
	}

	if 0 != padding {
		buf.buffer.PutInt64Ordered(tailIndex, MakeHeader(int32(padding), RecordDescriptor.PaddingMsgTypeID))
		tailIndex = 0
	}

	return tailIndex
}

func (buf *ManyToOne) checkMsgLength(length int32) {
	if length > buf.maxMsgLength {
		panic(fmt.Sprintf("encoded message exceeds maxMsgLength of %d, length=%d", buf.maxMsgLength, length))
	}
}

func (buf *ManyToOne) Write(msgTypeID int32, srcBuffer *atomic.Buffer, srcIndex int32, length int32) bool {

	isSuccessful := false

	CheckMsgTypeID(msgTypeID)
	buf.checkMsgLength(length)

	recordLength := length + RecordDescriptor.HeaderLength
	requiredCapacity := util.AlignInt32(recordLength, RecordDescriptor.RecordAlignment)
	recordIndex := buf.claimCapacity(requiredCapacity)

	if InsufficientCapacity != recordIndex {
		buf.buffer.PutInt64Ordered(recordIndex, MakeHeader(-recordLength, msgTypeID))
		buf.buffer.PutBytes(EncodedMsgOffset(recordIndex), srcBuffer, srcIndex, length)
		buf.buffer.PutInt32Ordered(LengthOffset(recordIndex), recordLength)

		isSuccessful = true
	}

	return isSuccessful
}

func (buf *ManyToOne) Read(Handler, messageCountLimit int) int32 {
	panic("Not implemented yet")
	return -1
}
