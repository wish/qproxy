// TODO: move elsewhere
package qproxy

// Copyright 2013 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import (
	"compress/gzip"
	"compress/zlib"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/golang/snappy"
)

const (
	acceptEncodingHeader  = "Accept-Encoding"
	contentEncodingHeader = "Content-Encoding"
	gzipEncoding          = "gzip"
	deflateEncoding       = "deflate"
	snappyEncoding        = "x-snappy-framed"
)

// We need to use sync.Pool to avoid LOTS of memory allocation
var gzipPool = sync.Pool{
	New: func() interface{} {
		w, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed)
		return w
	},
}

// We need to use sync.Pool to avoid LOTS of memory allocation
var deflatePool = sync.Pool{
	New: func() interface{} {
		w, _ := zlib.NewWriterLevel(nil, zlib.BestSpeed)
		return w
	},
}

// We need to use sync.Pool to avoid LOTS of memory allocation
var snappyPool = sync.Pool{
	New: func() interface{} {
		return snappy.NewWriter(nil)
	},
}

// Wrapper around http.Handler which adds suitable response compression based
// on the client's Accept-Encoding headers.
type compressedResponseWriter struct {
	http.ResponseWriter
	writer io.Writer
}

// Writes HTTP response content data.
func (c *compressedResponseWriter) Write(p []byte) (int, error) {
	return c.writer.Write(p)
}

func (c *compressedResponseWriter) Flush() {
	switch writerTyped := c.writer.(type) {
	case *gzip.Writer:
		writerTyped.Flush()
	case *zlib.Writer:
		writerTyped.Flush()
	case *snappy.Writer:
		writerTyped.Flush()
	}
}

// Closes the compressedResponseWriter and ensures to flush all data before.
func (c *compressedResponseWriter) Close() {
	c.Flush()
	if closer, ok := c.writer.(io.Closer); ok {
		closer.Close()
	}
	switch writerTyped := c.writer.(type) {
	case *gzip.Writer:
		gzipPool.Put(writerTyped)
	case *zlib.Writer:
		deflatePool.Put(writerTyped)
	case *snappy.Writer:
		snappyPool.Put(writerTyped)
	}
}

// Constructs a new compressedResponseWriter based on client request headers.
func newCompressedResponseWriter(writer http.ResponseWriter, req *http.Request) *compressedResponseWriter {
	encodings := strings.Split(req.Header.Get(acceptEncodingHeader), ",")
	for _, encoding := range encodings {
		switch strings.TrimSpace(encoding) {
		case gzipEncoding:
			writer.Header().Set(contentEncodingHeader, gzipEncoding)
			gw := gzipPool.Get().(*gzip.Writer)
			gw.Reset(writer)
			return &compressedResponseWriter{
				ResponseWriter: writer,
				writer:         gw,
			}
		case deflateEncoding:
			writer.Header().Set(contentEncodingHeader, deflateEncoding)
			dw := deflatePool.Get().(*zlib.Writer)
			dw.Reset(writer)
			return &compressedResponseWriter{
				ResponseWriter: writer,
				writer:         dw,
			}
		case snappyEncoding:
			writer.Header().Set(contentEncodingHeader, snappyEncoding)
			sw := snappyPool.Get().(*snappy.Writer)
			sw.Reset(writer)
			return &compressedResponseWriter{
				ResponseWriter: writer,
				writer:         sw,
			}
		}
	}

	// If we don't have one, return nothing
	return nil
}

// CompressionHandler is a wrapper around http.Handler which adds suitable
// response compression based on the client's Accept-Encoding headers.
type CompressionHandler struct {
	Handler func(http.ResponseWriter, *http.Request)
}

// ServeHTTP adds compression to the original http.Handler's ServeHTTP() method.
func (c CompressionHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	if compWriter := newCompressedResponseWriter(writer, req); compWriter == nil {
		c.Handler(writer, req)
	} else {
		c.Handler(compWriter, req)
		compWriter.Close()
	}
}
