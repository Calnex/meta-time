/*
Copyright (c) Facebook, Inc. and its affiliates.

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

package timestamp

import (
	"fmt"
	"net"
	"runtime"
	"testing"
	"time"

	"github.com/facebook/time/hostendian"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func reverse(s []byte) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func Test_byteToTime(t *testing.T) {
	timeb := []byte{63, 155, 21, 96, 0, 0, 0, 0, 52, 156, 191, 42, 0, 0, 0, 0}
	if hostendian.IsBigEndian {
		// reverse two int64 individually
		reverse(timeb[0:8])
		reverse(timeb[8:16])
	}
	res, err := byteToTime(timeb)
	require.Nil(t, err)

	require.Equal(t, int64(1612028735717200436), res.UnixNano())
}

func Test_ReadTXtimestamp(t *testing.T) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.Nil(t, err)
	defer conn.Close()

	connFd, err := ConnFd(conn)
	require.Nil(t, err)

	err = EnableSWTimestamps(connFd)
	require.Nil(t, err)

	start := time.Now()
	txts, attempts, err := ReadTXtimestamp(connFd)
	duration := time.Since(start)
	require.Equal(t, time.Time{}, txts)
	require.Equal(t, defaultTXTS, attempts)
	errStr := fmt.Sprintf("no TX timestamp found after %d tries", defaultTXTS)
	require.ErrorContains(t, err, errStr)
	require.GreaterOrEqual(t, duration, time.Duration(AttemptsTXTS)*TimeoutTXTS)

	AttemptsTXTS = 10
	TimeoutTXTS = 5 * time.Millisecond

	start = time.Now()
	txts, attempts, err = ReadTXtimestamp(connFd)
	duration = time.Since(start)
	require.Equal(t, time.Time{}, txts)
	require.Equal(t, 10, attempts)
	errStr = fmt.Sprintf("no TX timestamp found after %d tries", 10)
	require.ErrorContains(t, err, errStr)
	require.GreaterOrEqual(t, duration, time.Duration(AttemptsTXTS)*TimeoutTXTS)

	addr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
	_, err = conn.WriteTo([]byte{}, addr)
	require.Nil(t, err)
	txts, attempts, err = ReadTXtimestamp(connFd)

	require.NotEqual(t, time.Time{}, txts)
	require.Equal(t, 1, attempts)
	require.Nil(t, err)
}

func Test_scmDataToTime(t *testing.T) {
	hwData := []byte{
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		63, 155, 21, 96, 0, 0, 0, 0, 52, 156, 191, 42, 0, 0, 0, 0,
	}
	swData := []byte{
		63, 155, 21, 96, 0, 0, 0, 0, 52, 156, 191, 42, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}
	noData := []byte{
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}

	if hostendian.IsBigEndian {
		// make two int64 BigEndian
		reverse(hwData[32:40])
		reverse(hwData[40:48])
		// ditto, but different position of int64s
		reverse(swData[0:8])
		reverse(swData[8:16])
	}

	tests := []struct {
		name    string
		data    []byte
		want    int64
		wantErr bool
	}{
		{
			name:    "hardware timestamp",
			data:    hwData,
			want:    1612028735717200436,
			wantErr: false,
		},
		{
			name:    "software timestamp",
			data:    swData,
			want:    1612028735717200436,
			wantErr: false,
		},
		{
			name:    "zero timestamp",
			data:    noData,
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := scmDataToTime(tt.data)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.Nil(t, err)
				require.Equal(t, tt.want, res.UnixNano())
			}
		})
	}
}

func TestEnableSWTimestampsRx(t *testing.T) {
	// listen to incoming udp packets
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer conn.Close()

	connFd, err := ConnFd(conn)
	require.NoError(t, err)

	// Allow reading of kernel timestamps via socket
	err = EnableSWTimestampsRx(connFd)
	require.NoError(t, err)

	// Check that socket option is set
	timestampsEnabled, _ := unix.GetsockoptInt(connFd, unix.SOL_SOCKET, unix.SO_TIMESTAMPING)
	newTimestampsEnabled, _ := unix.GetsockoptInt(connFd, unix.SOL_SOCKET, unix.SO_TIMESTAMPING_NEW)

	// At least one of them should be set, which it > 0
	require.Greater(t, timestampsEnabled+newTimestampsEnabled, 0, "None of the socket options is set")
}

func TestEnableSWTimestamps(t *testing.T) {
	// listen to incoming udp packets
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer conn.Close()

	connFd, err := ConnFd(conn)
	require.NoError(t, err)

	// Allow reading of kernel timestamps via socket
	err = EnableSWTimestamps(connFd)
	require.NoError(t, err)

	// Check that socket option is set
	timestampsEnabled, _ := unix.GetsockoptInt(connFd, unix.SOL_SOCKET, unix.SO_TIMESTAMPING)
	newTimestampsEnabled, _ := unix.GetsockoptInt(connFd, unix.SOL_SOCKET, unix.SO_TIMESTAMPING_NEW)

	// At least one of them should be set, which it > 0
	require.Greater(t, timestampsEnabled+newTimestampsEnabled, 0, "None of the socket options is set")
}

func TestEnableTimestamps(t *testing.T) {
	// listen to incoming udp packets
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer conn.Close()

	connFd, err := ConnFd(conn)
	require.NoError(t, err)

	// SOFTWARE
	// Allow reading of kernel timestamps via socket
	err = EnableTimestamps(SW, connFd, "lo")
	require.NoError(t, err)

	// Check that socket option is set
	timestampsEnabled, _ := unix.GetsockoptInt(connFd, unix.SOL_SOCKET, unix.SO_TIMESTAMPING)
	newTimestampsEnabled, _ := unix.GetsockoptInt(connFd, unix.SOL_SOCKET, unix.SO_TIMESTAMPING_NEW)

	// At least one of them should be set, which it > 0
	require.Greater(t, timestampsEnabled+newTimestampsEnabled, 0, "None of the socket options is set")

	// SOFTWARE_RX
	// Allow reading of kernel timestamps via socket
	err = EnableTimestamps(SWRX, connFd, "lo")
	require.NoError(t, err)

	// Check that socket option is set
	timestampsEnabled, _ = unix.GetsockoptInt(connFd, unix.SOL_SOCKET, unix.SO_TIMESTAMPING)
	newTimestampsEnabled, _ = unix.GetsockoptInt(connFd, unix.SOL_SOCKET, unix.SO_TIMESTAMPING_NEW)

	// At least one of them should be set, which it > 0
	require.Greater(t, timestampsEnabled+newTimestampsEnabled, 0, "None of the socket options is set")

	// Unsupported
	err = EnableTimestamps(42, connFd, "lo")
	require.Equal(t, fmt.Errorf("Unrecognized timestamp type: Unsupported"), err)
}

func TestSocketControlMessageTimestamp(t *testing.T) {
	if timestamping != unix.SO_TIMESTAMPING_NEW {
		t.Skip("This test supports SO_TIMESTAMPING_NEW only. No sample of SO_TIMESTAMPING")
	}

	var b []byte
	var toob int

	// unix.Cmsghdr used in socketControlMessageTimestamp differs depending on platform
	switch runtime.GOARCH {
	case "amd64":
		b = []byte{
			0x40, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			0x1, 0x0, 0x0, 0x0, 0x41, 0x0, 0x0, 0x0,
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			0x79, 0xab, 0x24, 0x68, 0x0, 0x0, 0x0,
			0x0, 0xfc, 0xab, 0xf9, 0x8, 0x0, 0x0,
			0x0, 0x0, 0x3c, 0x0, 0x0, 0x0, 0x0, 0x0,
			0x0, 0x0, 0x29, 0x0, 0x0, 0x0, 0x19, 0x0,
			0x0, 0x0, 0x2a, 0x0, 0x0, 0x0, 0x4, 0x0,
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x0,
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			0x0, 0x0,
		}
		toob = len(b)
	default:
		t.Skip("This test checks amd64 platform only")
	}

	ts, err := socketControlMessageTimestamp(b, toob)
	require.NoError(t, err)
	require.Equal(t, int64(1747233657150580220), ts.UnixNano())
}

func TestSocketControlMessageTimestampFail(t *testing.T) {
	if timestamping != unix.SO_TIMESTAMPING_NEW {
		t.Skip("This test supports SO_TIMESTAMPING_NEW only. No sample of SO_TIMESTAMPING")
	}

	_, err := socketControlMessageTimestamp(make([]byte, 16), 16)
	require.ErrorIs(t, errNoTimestamp, err)
}

func TestReadPacketWithRXTimestamp(t *testing.T) {
	request := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 42}
	// listen to incoming udp packets
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("localhost"), Port: 0})
	require.NoError(t, err)
	defer conn.Close()

	// get connection file descriptor
	connFd, err := ConnFd(conn)
	require.NoError(t, err)

	// Allow reading of kernel timestamps via socket
	err = EnableSWTimestampsRx(connFd)
	require.NoError(t, err)

	err = unix.SetNonblock(connFd, false)
	require.NoError(t, err)

	// Send a client request
	timeout := 1 * time.Second
	cconn, err := net.DialTimeout("udp", conn.LocalAddr().String(), timeout)
	require.NoError(t, err)
	defer cconn.Close()
	_, err = cconn.Write(request)
	require.NoError(t, err)

	// read kernel timestamp from incoming packet
	data, returnaddr, nowKernelTimestamp, err := ReadPacketWithRXTimestamp(connFd)
	require.Equal(t, request, data, "We should have the same request arriving on the server")
	require.Equal(t, time.Now().Unix()/10, nowKernelTimestamp.Unix()/10, "kernel timestamps should be within 10s")
	requireEqualNetAddrSockAddr(t, cconn.LocalAddr(), returnaddr)
	require.NoError(t, err)
}

func TestReadPacketWithRXTXTimestamp(t *testing.T) {
	request := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 42}
	// listen to incoming udp packets
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("localhost"), Port: 0})
	require.NoError(t, err)
	defer conn.Close()

	// get connection file descriptor
	connFd, err := ConnFd(conn)
	require.NoError(t, err)

	// Allow reading of kernel timestamps via socket
	err = EnableSWTimestamps(connFd)
	require.NoError(t, err)

	err = unix.SetNonblock(connFd, false)
	require.NoError(t, err)

	// Send a client request
	timeout := 1 * time.Second
	cconn, err := net.DialTimeout("udp", conn.LocalAddr().String(), timeout)
	require.NoError(t, err)
	defer cconn.Close()
	_, err = cconn.Write(request)
	require.NoError(t, err)

	// read kernel timestamp from incoming packet
	data, returnaddr, nowKernelTimestamp, err := ReadPacketWithRXTimestamp(connFd)
	require.NoError(t, err)
	require.Equal(t, request, data, "We should have the same request arriving on the server")
	require.Equal(t, time.Now().Unix()/10, nowKernelTimestamp.Unix()/10, "kernel timestamps should be within 10s")
	requireEqualNetAddrSockAddr(t, cconn.LocalAddr(), returnaddr)

	// send packet and read TX timestamp
	_, err = conn.WriteTo(request, cconn.LocalAddr())
	require.NoError(t, err)
	txts, attempts, err := ReadTXtimestamp(connFd)
	require.NotEqual(t, time.Time{}, txts)
	require.Equal(t, 1, attempts)
	require.Nil(t, err)
}

func TestReadHWTimestampCaps(t *testing.T) {
	// listen to incoming udp packets
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("localhost"), Port: 0})
	require.NoError(t, err)
	defer conn.Close()

	// get connection file descriptor
	connFd, err := ConnFd(conn)
	require.NoError(t, err)

	rxFilters, txType, err := ioctlHWTimestampCaps(connFd, "lo")
	require.Error(t, err)
	// hw timestamps are disabled for lo
	require.Equal(t, int32(0), txType)
	require.Equal(t, int32(0), rxFilters)
}

func TestScmDataToSeqID(t *testing.T) {
	// 0x2a is ENOMSG - see /usr/include/asm-generic/errno.h
	hwData := []byte{
		0x2a, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x0, 0xd2, 0x4, 0x0, 0x0,
	}
	seqID, err := scmDataToSeqID(hwData)
	require.NoError(t, err)
	require.Equal(t, uint32(1234), seqID)
}

func TestScmDataToSeqIDErrornoNotENOMSG(t *testing.T) {
	// 0x26 is ENOSYS - see /usr/include/asm-generic/errno.h
	hwData := []byte{
		0x26, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x0, 0xd2, 0x4, 0x0, 0x0,
	}
	_, err := scmDataToSeqID(hwData)
	require.Error(t, err)
	require.ErrorContains(t, err, "Expected ENOMSG but got function not implemented")
}

func TestSeqIDSocketControlMessage(t *testing.T) {
	soob := make([]byte, unix.CmsgSpace(SizeofSeqID))
	seqID := uint32(8765)
	SeqIDSocketControlMessage(seqID, soob)
	// Socket Control Message with 20 byte length (0x14), level SOL_SOCKET (0x1)
	// and type SCM_TS_OPT_ID (0x51) and a 4 byte payload (0x3d, 0x22, 0x0, 0x0)
	sockControlMsg := []byte{
		0x14, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		0x1, 0x0, 0x0, 0x0, 0x51, 0x0, 0x0, 0x0,
		0x3d, 0x22, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	}
	require.Equal(t, sockControlMsg, soob)
}

func TestSocketControlMessageSeqIDTimestamp(t *testing.T) {
	tboob := 128
	seqID := uint32(3248)
	// byte array of 2 socket control messages: First message includes a timestamp and the second
	// message includes the sequence ID of a Sync packet
	// cmsghdr struct comprises len (8 bytes), level (4 bytes), type (4 bytes) and data (length can vary)
	sockControlMsgs := []byte{
		0x40, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, // message has len of 64 bytes (0x40), level SOL_SOCKET (0x1), type SO_TIMESTAMPING_NEW (0x41)
		0x1, 0x0, 0x0, 0x0, 0x41, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		0x69, 0x75, 0x23, 0x68, 0x0, 0x0, 0x0, 0x0,
		0x7b, 0xcb, 0x4, 0x6, 0x0, 0x0, 0x0, 0x0,
		0x3c, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, // message has len of 60 bytes (0x3c) level SOL_IPV6 (0x29) and type IPV6_RECVERR (0x19)
		0x29, 0x0, 0x0, 0x0, 0x19, 0x0, 0x0, 0x0,
		0x2a, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0, // sock_extended_err with errno ENOMSG (0x2a) and data field of 4 bytes (0x4) which is the Sequence ID of 3248 (0xb0, 0x0c, 0x0, 0x0)
		0x0, 0x0, 0x0, 0x0, 0xb0, 0x0c, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, // Last 4 bytes are there to align the socket message to an 8-byte boundary
	}
	ts, err := socketControlMessageSeqIDTimestamp(sockControlMsgs, tboob, seqID)
	require.NoError(t, err)
	require.Equal(t, int64(1747154281100977531), ts.UnixNano())
}
