// Copyright 2016 The Prometheus Authors
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

package main

import (
	"bytes"
	"log"
	"net/http"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"

	"github.com/prometheus/prometheus/model/histogram"
	"github.com/prometheus/prometheus/prompb"
)

var testHistogram = histogram.Histogram{
	Schema:          2,
	ZeroThreshold:   1e-128,
	ZeroCount:       0,
	Count:           0,
	Sum:             20,
	PositiveSpans:   []histogram.Span{{Offset: 0, Length: 1}},
	PositiveBuckets: []int64{1},
	NegativeSpans:   []histogram.Span{{Offset: 0, Length: 1}},
	NegativeBuckets: []int64{-1},
}

var writeRequest = &prompb.WriteRequest{
	Timeseries: []prompb.TimeSeries{
		{
			Labels: []prompb.Label{
				{Name: "__name__", Value: "test_metric1"},
				{Name: "b", Value: "c"},
				{Name: "baz", Value: "qux"},
				{Name: "d", Value: "e"},
				{Name: "foo", Value: "bar"},
			},
			Samples:   []prompb.Sample{{Value: 1, Timestamp: 0}},
			Exemplars: []prompb.Exemplar{{Labels: []prompb.Label{{Name: "f", Value: "g"}}, Value: 1, Timestamp: 0}},
			// Histograms: []prompb.Histogram{remote.HistogramToHistogramProto(0, &testHistogram), remote.FloatHistogramToHistogramProto(1, testHistogram.ToFloat())},
		},
		{
			Labels: []prompb.Label{
				{Name: "__name__", Value: "test_metric1"},
				{Name: "b", Value: "c"},
				{Name: "baz", Value: "qux"},
				{Name: "d", Value: "e"},
				{Name: "foo", Value: "bar"},
			},
			Samples:   []prompb.Sample{{Value: 2, Timestamp: 1}},
			Exemplars: []prompb.Exemplar{{Labels: []prompb.Label{{Name: "h", Value: "i"}}, Value: 2, Timestamp: 1}},
			// Histograms: []prompb.Histogram{remote.HistogramToHistogramProto(2, &testHistogram), remote.FloatHistogramToHistogramProto(3, testHistogram.ToFloat())},
		},
	},
}

func buildWriteRequest(samples []prompb.TimeSeries, metadata []prompb.MetricMetadata, pBuf *proto.Buffer, buf []byte) ([]byte, int64, error) {
	var highest int64
	for _, ts := range samples {
		// At the moment we only ever append a TimeSeries with a single sample or exemplar in it.
		if len(ts.Samples) > 0 && ts.Samples[0].Timestamp > highest {
			highest = ts.Samples[0].Timestamp
		}
		if len(ts.Exemplars) > 0 && ts.Exemplars[0].Timestamp > highest {
			highest = ts.Exemplars[0].Timestamp
		}
		if len(ts.Histograms) > 0 && ts.Histograms[0].Timestamp > highest {
			highest = ts.Histograms[0].Timestamp
		}
	}

	req := &prompb.WriteRequest{
		Timeseries: samples,
		Metadata:   metadata,
	}

	if pBuf == nil {
		pBuf = proto.NewBuffer(nil) // For convenience in tests. Not efficient.
	} else {
		pBuf.Reset()
	}

	err := pBuf.Marshal(req)
	if err != nil {
		return nil, highest, err
	}

	// snappy uses len() to see if it needs to allocate a new slice. Make the
	// buffer as long as possible.
	if buf != nil {
		buf = buf[0:cap(buf)]
	}

	compressed := snappy.Encode(buf, pBuf.Bytes())
	return compressed, highest, nil
}

func main() {
	url := "http://127.0.0.1:1234/receive"

	buf, _, err := buildWriteRequest(writeRequest.Timeseries, nil, nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(buf))
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()
}
