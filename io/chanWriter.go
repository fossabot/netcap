/*
 * NETCAP - Traffic Analysis Framework
 * Copyright (c) 2017-2020 Philipp Mieden <dreadl0ck [at] protonmail [dot] ch>
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package io

import (
	"bufio"
	"log"
	"os"
	"runtime"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/klauspost/pgzip"

	"github.com/dreadl0ck/netcap/defaults"
	"github.com/dreadl0ck/netcap/delimited"
	"github.com/dreadl0ck/netcap/types"
)

// ChanWriter writes length delimited, serialized protobuf records into a channel.
type ChanWriter struct {
	bWriter *bufio.Writer
	gWriter *pgzip.Writer
	dWriter *delimited.Writer
	cWriter *ChanProtoWriter

	file *os.File
	mu   sync.Mutex
	wc   *WriterConfig
}

// NewChanWriter initializes and configures a new ChanWriter instance.
func NewChanWriter(wc *WriterConfig) *ChanWriter {
	w := &ChanWriter{}
	w.wc = wc

	if wc.MemBufferSize <= 0 {
		wc.MemBufferSize = defaults.BufferSize
	}

	if wc.Buffer || wc.Compress {
		panic("buffering or compression cannot be activated when running using writeChan")
	}

	w.cWriter = NewChanProtoWriter(wc.ChanSize)

	// buffer data?
	if wc.Buffer {
		if wc.Compress {
			// experiment: pgzip -> file
			var errGzipWriter error
			w.gWriter, errGzipWriter = pgzip.NewWriterLevel(w.file, wc.CompressionLevel)

			if errGzipWriter != nil {
				panic(errGzipWriter)
			}
			// experiment: buffer -> pgzip
			w.bWriter = bufio.NewWriterSize(w.gWriter, wc.MemBufferSize)
			// experiment: delimited -> buffer
			w.dWriter = delimited.NewWriter(w.bWriter)
		} else {
			w.bWriter = bufio.NewWriterSize(w.file, wc.MemBufferSize)
			w.dWriter = delimited.NewWriter(w.bWriter)
		}
	} else {
		if wc.Compress {
			var errGzipWriter error
			w.gWriter, errGzipWriter = pgzip.NewWriterLevel(w.file, wc.CompressionLevel)
			if errGzipWriter != nil {
				panic(errGzipWriter)
			}
			w.dWriter = delimited.NewWriter(w.gWriter)
		} else {
			// write into channel writer without compression
			w.dWriter = delimited.NewWriter(w.cWriter)
		}
	}

	if w.gWriter != nil {
		// To get any performance gains, you should at least be compressing more than 1 megabyte of data at the time.
		// You should at least have a block size of 100k and at least a number of blocks that match the number of cores
		// you would like to utilize, but about twice the number of blocks would be the best.
		if err := w.gWriter.SetConcurrency(wc.CompressionBlockSize, runtime.GOMAXPROCS(0)*2); err != nil {
			log.Fatal("failed to configure compression package: ", err)
		}
	}

	return w
}

// WriteProto writes a protobuf message.
func (w *ChanWriter) Write(msg proto.Message) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = w.cWriter.Write(data)

	return err
}

// WriteHeader writes a netcap file header for protobuf encoded audit record files.
func (w *ChanWriter) WriteHeader(t types.Type) error {
	data, err := proto.Marshal(NewHeader(t, w.wc.Source, w.wc.Version, w.wc.IncludesPayloads, w.wc.StartTime))
	if err != nil {
		return err
	}

	_, err = w.cWriter.Write(data)

	return err
}

// Close flushes and closes the writer and the associated file handles.
func (w *ChanWriter) Close() (name string, size int64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.wc.Buffer {
		flushWriters(w.bWriter)
	}

	if w.wc.Compress {
		closeGzipWriters(w.gWriter)
	}

	return closeFile(w.wc.Out, w.file, w.wc.Name)
}

// GetChan returns a channel for receiving bytes.
func (w *ChanWriter) GetChan() <-chan []byte {
	return w.cWriter.Chan()
}

// ChanProtoWriter writes into a []byte chan.
type ChanProtoWriter struct {
	ch chan []byte
}

// NewChanProtoWriter returns a new channel proto writer instance.
func NewChanProtoWriter(size int) *ChanProtoWriter {
	return &ChanProtoWriter{make(chan []byte, size)}
}

// Chan returns the byte channel used to receive data.
func (w *ChanProtoWriter) Chan() <-chan []byte {
	return w.ch
}

// WriteRecord writes a protocol buffer into the channel writer.
func (w *ChanProtoWriter) Write(p []byte) (int, error) {
	w.ch <- p

	return len(p), nil
}

// Close will close the channel writer.
func (w *ChanProtoWriter) Close() error {
	close(w.ch)

	return nil
}
