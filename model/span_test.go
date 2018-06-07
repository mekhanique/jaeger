// Copyright (c) 2017 Uber Technologies, Inc.
//
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

package model_test

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jaegertracing/jaeger/model"
)

var (
	_ jsonpb.JSONPBUnmarshaler = new(model.TraceID)
	_ jsonpb.JSONPBMarshaler   = new(model.TraceID)
	// _ jsonpb.JSONPBUnmarshaler = new(model.SpanID)
	// _ jsonpb.JSONPBMarshaler   = new(model.SpanID)
)

func TestTraceIDMarshalJSONPB(t *testing.T) {
	testCases := []struct {
		hi, lo uint64
		out    string
	}{
		{lo: 1, out: `"1"`},
		{lo: 15, out: `"f"`},
		{lo: 31, out: `"1f"`},
		{lo: 257, out: `"101"`},
		{hi: 1, lo: 1, out: `"10000000000000001"`},
		{hi: 257, lo: 1, out: `"1010000000000000001"`},
	}
	for _, testCase := range testCases {
		id := model.NewTraceID(testCase.hi, testCase.lo)
		out := new(bytes.Buffer)
		err := new(jsonpb.Marshaler).Marshal(out, &id)
		if assert.NoError(t, err) {
			assert.Equal(t, testCase.out, out.String())
		}
	}
}

func TestTraceIDUnmarshalJSONPB(t *testing.T) {
	testCases := []struct {
		in     string
		hi, lo uint64
		err    bool
	}{
		{lo: 1, in: `"1"`},
		{lo: 15, in: `"f"`},
		{lo: 31, in: `"1f"`},
		{lo: 257, in: `"101"`},
		{hi: 1, lo: 1, in: `"10000000000000001"`},
		{hi: 257, lo: 1, in: `"1010000000000000001"`},
		{err: true, in: ``},
		{err: true, in: `"x"`},
		{err: true, in: `"x0000000000000001"`},
		{err: true, in: `"1x000000000000001"`},
		{err: true, in: `"10123456789abcdef0123456789abcdef"`},
	}
	for _, testCase := range testCases {
		var id model.TraceID
		err := jsonpb.Unmarshal(bytes.NewReader([]byte(testCase.in)), &id)
		if testCase.err {
			assert.Error(t, err)
		} else {
			if assert.NoError(t, err) {
				assert.Equal(t, testCase.hi, id.High)
				assert.Equal(t, testCase.lo, id.Low)
			}
		}
	}
	// for code coverage
	var id model.TraceID
	err := id.UnmarshalJSONPB(nil, []byte(""))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TraceID JSON string cannot be shorter than 3 chars")
	err = id.UnmarshalJSONPB(nil, []byte("123"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TraceID JSON string must be enclosed in quotes")
	_, err = id.MarshalText()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported method")
	err = id.UnmarshalText(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported method")
}

func TestSpanIDMarshalJSON(t *testing.T) {
	max := int64(-1)
	testCases := []struct {
		id  uint64
		out string
	}{
		{id: 1, out: "1"},
		{id: 15, out: "f"},
		{id: 31, out: "1f"},
		{id: 257, out: "101"},
		{id: uint64(max), out: "ffffffffffffffff"},
	}
	for _, testCase := range testCases {
		expected := fmt.Sprintf(`{"traceId":"0","spanId":"%s"}`, testCase.out)
		t.Run(expected, func(t *testing.T) {
			ref := &model.SpanRef{SpanID: model.SpanID(testCase.id)}
			out := new(bytes.Buffer)
			err := new(jsonpb.Marshaler).Marshal(out, ref)
			if assert.NoError(t, err) {
				assert.Equal(t, expected, out.String())
			}
		})
	}
}

func TestSpanIDUnmarshalJSON(t *testing.T) {
	testCases := []struct {
		in  string
		id  model.SpanID
		err bool
	}{
		{id: 1, in: "1"},
		{id: 15, in: "f"},
		{id: 31, in: "1f"},
		{id: 257, in: "101"},
		{err: true, in: ""},
		{err: true, in: "x"},
		{err: true, in: "x123"},
		{err: true, in: "10123456789abcdef"},
	}
	for _, testCase := range testCases {
		in := fmt.Sprintf(`{"traceId":"0","spanId":"%s"}`, testCase.in)
		t.Run(in, func(t *testing.T) {
			var ref model.SpanRef
			err := jsonpb.Unmarshal(bytes.NewReader([]byte(in)), &ref)
			if testCase.err {
				assert.Error(t, err)
			} else {
				if assert.NoError(t, err) {
					assert.Equal(t, testCase.id, ref.SpanID)
				}
			}
		})
	}
	// for code coverage
	var id model.SpanID
	err := id.UnmarshalJSONPB(nil, []byte(""))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SpanID JSON string cannot be shorter than 3 chars")
	err = id.UnmarshalJSONPB(nil, []byte("123"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SpanID JSON string must be enclosed in quotes")
}

func TestIsRPCClientServer(t *testing.T) {
	span1 := &model.Span{
		Tags: model.KeyValues{
			model.String(string(ext.SpanKind), string(ext.SpanKindRPCClientEnum)),
		},
	}
	assert.True(t, span1.IsRPCClient())
	assert.False(t, span1.IsRPCServer())
	span2 := &model.Span{}
	assert.False(t, span2.IsRPCClient())
	assert.False(t, span2.IsRPCServer())
}

func TestIsDebug(t *testing.T) {
	flags := model.Flags(0)
	flags.SetDebug()
	assert.True(t, flags.IsDebug())
	flags = model.Flags(0)
	assert.False(t, flags.IsDebug())

	flags = model.Flags(32)
	assert.False(t, flags.IsDebug())
	flags.SetDebug()
	assert.True(t, flags.IsDebug())
}

func TestIsSampled(t *testing.T) {
	flags := model.Flags(0)
	flags.SetSampled()
	assert.True(t, flags.IsSampled())
	flags = model.Flags(0)
	flags.SetDebug()
	assert.False(t, flags.IsSampled())
}

func TestSpanHash(t *testing.T) {
	kvs := model.KeyValues{
		model.String("x", "y"),
		model.String("x", "y"),
		model.String("x", "z"),
	}
	spans := make([]*model.Span, len(kvs))
	codes := make([]uint64, len(kvs))
	// create 3 spans that are only different in some KeyValues
	for i := range kvs {
		spans[i] = makeSpan(kvs[i])
		hc, err := model.HashCode(spans[i])
		require.NoError(t, err)
		codes[i] = hc
	}
	assert.Equal(t, codes[0], codes[1])
	assert.NotEqual(t, codes[0], codes[2])
}

func TestParentSpanID(t *testing.T) {
	span := makeSpan(model.String("k", "v"))
	assert.Equal(t, model.NewSpanID(123), span.ParentSpanID())

	span.References = []model.SpanRef{
		model.NewFollowsFromRef(span.TraceID, model.NewSpanID(777)),
		model.NewChildOfRef(span.TraceID, model.NewSpanID(888)),
	}
	assert.Equal(t, model.NewSpanID(888), span.ParentSpanID())

	span.References = []model.SpanRef{
		model.NewChildOfRef(model.NewTraceID(321, 0), model.NewSpanID(999)),
	}
	assert.Equal(t, model.NewSpanID(0), span.ParentSpanID())
}

func TestReplaceParentSpanID(t *testing.T) {
	span := makeSpan(model.String("k", "v"))
	assert.Equal(t, model.NewSpanID(123), span.ParentSpanID())

	span.ReplaceParentID(789)
	assert.Equal(t, model.NewSpanID(789), span.ParentSpanID())

	span.References = []model.SpanRef{
		model.NewChildOfRef(model.NewTraceID(321, 0), model.NewSpanID(999)),
	}
	span.ReplaceParentID(789)
	assert.Equal(t, model.NewSpanID(789), span.ParentSpanID())
}

func makeSpan(someKV model.KeyValue) *model.Span {
	traceID := model.NewTraceID(0, 123)
	return &model.Span{
		TraceID:       traceID,
		SpanID:        model.NewSpanID(567),
		OperationName: "hi",
		References:    []model.SpanRef{model.NewChildOfRef(traceID, model.NewSpanID(123))},
		StartTime:     time.Unix(0, 1000),
		Duration:      5000,
		Tags:          model.KeyValues{someKV},
		Logs: []model.Log{
			{
				Timestamp: time.Unix(0, 1000),
				Fields:    model.KeyValues{someKV},
			},
		},
		Process: &model.Process{
			ServiceName: "xyz",
			Tags:        model.KeyValues{someKV},
		},
	}
}

// BenchmarkSpanHash-8   	   50000	     26977 ns/op	    2203 B/op	      68 allocs/op
func BenchmarkSpanHash(b *testing.B) {
	span := makeSpan(model.String("x", "y"))
	buf := &bytes.Buffer{}
	for i := 0; i < b.N; i++ {
		buf.Reset()
		span.Hash(buf)
	}
}
