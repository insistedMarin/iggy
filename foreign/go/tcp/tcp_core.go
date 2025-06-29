// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package tcp

import (
	"context"
	"encoding/binary"
	"net"
	"sync"
	"time"

	. "github.com/apache/iggy/foreign/go/contracts"
	iggcon "github.com/apache/iggy/foreign/go/contracts"
	ierror "github.com/apache/iggy/foreign/go/errors"
)

type IggyTcpClient struct {
	client             *net.TCPConn
	mtx                sync.Mutex
	MessageCompression iggcon.IggyMessageCompression
}

const (
	InitialBytesLength   = 4
	ExpectedResponseSize = 8
	MaxStringLength      = 255
)

func NewTcpMessageStream(
	ctx context.Context,
	url string,
	compression iggcon.IggyMessageCompression,
	heartbeatInterval time.Duration,
) (*IggyTcpClient, error) {
	addr, err := net.ResolveTCPAddr("tcp", url)
	if err != nil {
		return nil, err
	}

	var d = net.Dialer{
		KeepAlive: -1,
	}
	conn, err := d.DialContext(ctx, "tcp", addr.String())
	if err != nil {
		return nil, err
	}

	client := &IggyTcpClient{client: conn.(*net.TCPConn), MessageCompression: compression}

	if heartbeatInterval > 0 {
		go func() {
			ticker := time.NewTicker(heartbeatInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					client.Ping()
				}
			}
		}()
	}

	return client, nil
}

func (tms *IggyTcpClient) read(expectedSize int) (int, []byte, error) {
	var totalRead int
	buffer := make([]byte, expectedSize)

	for totalRead < expectedSize {
		readSize := expectedSize - totalRead
		n, err := tms.client.Read(buffer[totalRead : totalRead+readSize])
		if err != nil {
			return totalRead, buffer[:totalRead], err
		}
		totalRead += n
	}

	return totalRead, buffer, nil
}

func (tms *IggyTcpClient) write(payload []byte) (int, error) {
	var totalWritten int
	for totalWritten < len(payload) {
		n, err := tms.client.Write(payload[totalWritten:])
		if err != nil {
			return totalWritten, err
		}
		totalWritten += n
	}

	return totalWritten, nil
}

func (tms *IggyTcpClient) sendAndFetchResponse(message []byte, command CommandCode) ([]byte, error) {
	tms.mtx.Lock()
	defer tms.mtx.Unlock()

	payload := createPayload(message, command)
	if _, err := tms.write(payload); err != nil {
		return nil, err
	}

	_, buffer, err := tms.read(ExpectedResponseSize)
	if err != nil {
		return nil, err
	}

	length := int(binary.LittleEndian.Uint32(buffer[4:]))
	if responseCode := getResponseCode(buffer); responseCode != 0 {
		// TEMP: See https://github.com/apache/iggy/pull/604 for context.
		// from: https://github.com/apache/iggy/blob/master/sdk/src/tcp/client.rs#L326
		if responseCode == 2012 ||
			responseCode == 2013 ||
			responseCode == 1011 ||
			responseCode == 1012 ||
			responseCode == 46 ||
			responseCode == 51 ||
			responseCode == 5001 ||
			responseCode == 5004 {
			// do nothing
		} else {
			return nil, ierror.MapFromCode(responseCode)
		}

		return buffer, ierror.MapFromCode(responseCode)
	}

	if length <= 1 {
		return []byte{}, nil
	}

	_, buffer, err = tms.read(length)
	if err != nil {
		return nil, err
	}

	return buffer, nil
}

func createPayload(message []byte, command CommandCode) []byte {
	messageLength := len(message) + 4
	messageBytes := make([]byte, InitialBytesLength+messageLength)
	binary.LittleEndian.PutUint32(messageBytes[:4], uint32(messageLength))
	binary.LittleEndian.PutUint32(messageBytes[4:8], uint32(command))
	copy(messageBytes[8:], message)
	return messageBytes
}

func getResponseCode(buffer []byte) int {
	return int(binary.LittleEndian.Uint32(buffer[:4]))
}

func getResponseLength(buffer []byte) (int, error) {
	length := int(binary.LittleEndian.Uint32(buffer[4:]))
	if length <= 1 {
		return 0, &ierror.IggyError{
			Code:    0,
			Message: "Received empty response.",
		}
	}
	return length, nil
}
