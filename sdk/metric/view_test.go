// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metric // import "go.opentelemetry.io/otel/sdk/metric"

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/unit"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
)

var (
	schemaURL  = "https://opentelemetry.io/schemas/1.0.0"
	completeIP = Instrument{
		Name:        "foo",
		Description: "foo desc",
		Kind:        InstrumentKindSyncCounter,
		Unit:        unit.Bytes,
		Scope: instrumentation.Scope{
			Name:      "TestNewViewMatch",
			Version:   "v0.1.0",
			SchemaURL: schemaURL,
		},
	}
)

func scope(name, ver, url string) instrumentation.Scope {
	return instrumentation.Scope{Name: name, Version: ver, SchemaURL: url}
}

func testNewViewMatchName() func(t *testing.T) {
	tests := []struct {
		name     string
		criteria string
		match    []string
		notMatch []string
	}{
		{
			name:     "Exact",
			criteria: "foo",
			match:    []string{"foo"},
			notMatch: []string{"", "bar", "foobar", "barfoo", "ffooo"},
		},
		{
			name:     "Wildcard/*",
			criteria: "*",
			match:    []string{"", "foo", "foobar", "barfoo", "barfoobaz"},
		},
		{
			name:     "Wildcard/Front?",
			criteria: "?oo",
			match:    []string{"foo", "1oo"},
			notMatch: []string{"", "bar", "foobar", "barfoo", "barfoobaz"},
		},
		{
			name:     "Wildcard/Back?",
			criteria: "fo?",
			match:    []string{"foo", "fo1"},
			notMatch: []string{"", "bar", "foobar", "barfoo", "barfoobaz"},
		},
		{
			name:     "Wildcard/Front*",
			criteria: "*foo",
			match:    []string{"foo", "123foo", "barfoo"},
			notMatch: []string{"", "bar", "foobar", "barfoobaz"},
		},
		{
			name:     "Wildcard/Back*",
			criteria: "foo*",
			match:    []string{"foo", "foo1", "foobar"},
			notMatch: []string{"", "bar", "barfoo", "barfoobaz"},
		},
		{
			name:     "Wildcard/FrontBack*",
			criteria: "*foo*",
			match:    []string{"foo", "foo1", "1foo", "1foo1", "foobar", "barfoobaz"},
			notMatch: []string{"", "bar"},
		},
		{
			name:     "Wildcard/Front**",
			criteria: "**foo",
			match:    []string{"foo", "123foo", "barfoo", "afoo"},
			notMatch: []string{"", "bar", "foobar", "barfoobaz"},
		},
		{
			name:     "Wildcard/Back**",
			criteria: "foo**",
			match:    []string{"foo", "foo1", "fooa", "foobar"},
			notMatch: []string{"", "bar", "barfoo", "barfoobaz"},
		},
		{
			name:     "Wildcard/Front*?",
			criteria: "*?oo",
			match:    []string{"foo", "123foo", "barfoo", "afoo"},
			notMatch: []string{"", "fo", "bar", "foobar", "barfoobaz"},
		},
		{
			name:     "Wildcard/Back*?",
			criteria: "fo*?",
			match:    []string{"foo", "foo1", "fooa", "foobar"},
			notMatch: []string{"", "bar", "barfoo", "barfoobaz"},
		},
		{
			name:     "Wildcard/Front?*",
			criteria: "?*oo",
			match:    []string{"foo", "123foo", "barfoo", "afoo"},
			notMatch: []string{"", "oo", "fo", "bar", "foobar", "barfoobaz"},
		},
		{
			name:     "Wildcard/Back?*",
			criteria: "fo?*",
			match:    []string{"foo", "foo1", "fooa", "foobar"},
			notMatch: []string{"", "fo", "bar", "barfoo", "barfoobaz"},
		},
		{
			name:     "Wildcard/Middle*",
			criteria: "f*o",
			match:    []string{"fo", "foo", "fooo", "fo12baro"},
			notMatch: []string{"", "bar", "barfoo", "barfoobaz"},
		},
		{
			name:     "Wildcard/Middle?",
			criteria: "f?o",
			match:    []string{"foo", "f1o"},
			notMatch: []string{"", "fo", "fooo", "fo12baro", "bar"},
		},
		{
			name:     "Wildcard/MetaCharacters",
			criteria: "*.+()|[]{}^$-_?",
			match:    []string{"aa.+()|[]{}^$-_b", ".+()|[]{}^$-_b"},
			notMatch: []string{"", "foo", ".+()|[]{}^$-_"},
		},
	}

	return func(t *testing.T) {
		for _, test := range tests {
			v := NewView(Instrument{Name: test.criteria}, Stream{})
			t.Run(test.name, func(t *testing.T) {
				for _, n := range test.match {
					_, matches := v(Instrument{Name: n})
					assert.Truef(t, matches, "%s does not match %s", test.criteria, n)
				}
				for _, n := range test.notMatch {
					_, matches := v(Instrument{Name: n})
					assert.Falsef(t, matches, "%s matches %s", test.criteria, n)
				}
			})
		}
	}
}

func TestNewViewMatch(t *testing.T) {
	// Avoid boilerplate for name match testing.
	t.Run("Name", testNewViewMatchName())

	tests := []struct {
		name       string
		criteria   Instrument
		matches    []Instrument
		notMatches []Instrument
	}{
		{
			name:       "Empty",
			notMatches: []Instrument{{}, {Name: "foo"}, completeIP},
		},
		{
			name:       "Description",
			criteria:   Instrument{Description: "foo desc"},
			matches:    []Instrument{{Description: "foo desc"}, completeIP},
			notMatches: []Instrument{{}, {Description: "foo"}, {Description: "desc"}},
		},
		{
			name:     "Kind",
			criteria: Instrument{Kind: InstrumentKindSyncCounter},
			matches:  []Instrument{{Kind: InstrumentKindSyncCounter}, completeIP},
			notMatches: []Instrument{
				{},
				{Kind: InstrumentKindSyncUpDownCounter},
				{Kind: InstrumentKindSyncHistogram},
				{Kind: InstrumentKindAsyncCounter},
				{Kind: InstrumentKindAsyncUpDownCounter},
				{Kind: InstrumentKindAsyncGauge},
			},
		},
		{
			name:     "Unit",
			criteria: Instrument{Unit: unit.Bytes},
			matches:  []Instrument{{Unit: unit.Bytes}, completeIP},
			notMatches: []Instrument{
				{},
				{Unit: unit.Dimensionless},
				{Unit: unit.Unit("K")},
			},
		},
		{
			name:     "ScopeName",
			criteria: Instrument{Scope: scope("TestNewViewMatch", "", "")},
			matches: []Instrument{
				{Scope: scope("TestNewViewMatch", "", "")},
				completeIP,
			},
			notMatches: []Instrument{
				{},
				{Scope: scope("PrefixTestNewViewMatch", "", "")},
				{Scope: scope("TestNewViewMatchSuffix", "", "")},
				{Scope: scope("alt", "", "")},
			},
		},
		{
			name:     "ScopeVersion",
			criteria: Instrument{Scope: scope("", "v0.1.0", "")},
			matches: []Instrument{
				{Scope: scope("", "v0.1.0", "")},
				completeIP,
			},
			notMatches: []Instrument{
				{},
				{Scope: scope("", "v0.1.0-RC1", "")},
				{Scope: scope("", "v0.1.1", "")},
			},
		},
		{
			name:     "ScopeSchemaURL",
			criteria: Instrument{Scope: scope("", "", schemaURL)},
			matches: []Instrument{
				{Scope: scope("", "", schemaURL)},
				completeIP,
			},
			notMatches: []Instrument{
				{},
				{Scope: scope("", "", schemaURL+"/path")},
				{Scope: scope("", "", "https://go.dev")},
			},
		},
		{
			name:     "Scope",
			criteria: Instrument{Scope: scope("TestNewViewMatch", "v0.1.0", schemaURL)},
			matches: []Instrument{
				{Scope: scope("TestNewViewMatch", "v0.1.0", schemaURL)},
				completeIP,
			},
			notMatches: []Instrument{
				{},
				{Scope: scope("CompleteMisMatch", "v0.2.0", "https://go.dev")},
				{Scope: scope("NameMisMatch", "v0.1.0", schemaURL)},
			},
		},
		{
			name:     "Complete",
			criteria: completeIP,
			matches:  []Instrument{completeIP},
			notMatches: []Instrument{
				{},
				{Name: "foo"},
				{
					Name:        "Wrong Name",
					Description: "foo desc",
					Kind:        InstrumentKindSyncCounter,
					Unit:        unit.Bytes,
					Scope:       scope("TestNewViewMatch", "v0.1.0", schemaURL),
				},
				{
					Name:        "foo",
					Description: "Wrong Description",
					Kind:        InstrumentKindSyncCounter,
					Unit:        unit.Bytes,
					Scope:       scope("TestNewViewMatch", "v0.1.0", schemaURL),
				},
				{
					Name:        "foo",
					Description: "foo desc",
					Kind:        InstrumentKindAsyncUpDownCounter,
					Unit:        unit.Bytes,
					Scope:       scope("TestNewViewMatch", "v0.1.0", schemaURL),
				},
				{
					Name:        "foo",
					Description: "foo desc",
					Kind:        InstrumentKindSyncCounter,
					Unit:        unit.Dimensionless,
					Scope:       scope("TestNewViewMatch", "v0.1.0", schemaURL),
				},
				{
					Name:        "foo",
					Description: "foo desc",
					Kind:        InstrumentKindSyncCounter,
					Unit:        unit.Bytes,
					Scope:       scope("Wrong Scope Name", "v0.1.0", schemaURL),
				},
				{
					Name:        "foo",
					Description: "foo desc",
					Kind:        InstrumentKindSyncCounter,
					Unit:        unit.Bytes,
					Scope:       scope("TestNewViewMatch", "v1.4.3", schemaURL),
				},
				{
					Name:        "foo",
					Description: "foo desc",
					Kind:        InstrumentKindSyncCounter,
					Unit:        unit.Bytes,
					Scope:       scope("TestNewViewMatch", "v0.1.0", "https://go.dev"),
				},
			},
		},
	}

	for _, test := range tests {
		v := NewView(test.criteria, Stream{})
		t.Run(test.name, func(t *testing.T) {
			for _, instrument := range test.matches {
				_, matches := v(instrument)
				assert.Truef(t, matches, "view does not match %#v", instrument)
			}

			for _, instrument := range test.notMatches {
				_, matches := v(instrument)
				assert.Falsef(t, matches, "view matches %#v", instrument)
			}
		})
	}
}

func TestNewViewReplace(t *testing.T) {
	alt := "alternative value"
	tests := []struct {
		name string
		mask Stream
		want func(Instrument) Stream
	}{
		{
			name: "Nothing",
			want: func(i Instrument) Stream {
				return Stream{
					Name:        i.Name,
					Description: i.Description,
					Unit:        i.Unit,
				}
			},
		},
		{
			name: "Name",
			mask: Stream{Name: alt},
			want: func(i Instrument) Stream {
				return Stream{
					Name:        alt,
					Description: i.Description,
					Unit:        i.Unit,
				}
			},
		},
		{
			name: "Description",
			mask: Stream{Description: alt},
			want: func(i Instrument) Stream {
				return Stream{
					Name:        i.Name,
					Description: alt,
					Unit:        i.Unit,
				}
			},
		},
		{
			name: "Unit",
			mask: Stream{Unit: unit.Dimensionless},
			want: func(i Instrument) Stream {
				return Stream{
					Name:        i.Name,
					Description: i.Description,
					Unit:        unit.Dimensionless,
				}
			},
		},
		{
			name: "Aggregation",
			mask: Stream{Aggregation: aggregation.LastValue{}},
			want: func(i Instrument) Stream {
				return Stream{
					Name:        i.Name,
					Description: i.Description,
					Unit:        i.Unit,
					Aggregation: aggregation.LastValue{},
				}
			},
		},
		{
			name: "Complete",
			mask: Stream{
				Name:        alt,
				Description: alt,
				Unit:        unit.Dimensionless,
				Aggregation: aggregation.LastValue{},
			},
			want: func(i Instrument) Stream {
				return Stream{
					Name:        alt,
					Description: alt,
					Unit:        unit.Dimensionless,
					Aggregation: aggregation.LastValue{},
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, match := NewView(completeIP, test.mask)(completeIP)
			require.True(t, match, "view did not match exact criteria")
			assert.Equal(t, test.want(completeIP), got)
		})
	}

	// Go does not allow for the comparison of function values, even their
	// addresses. Therefore, the AttributeFilter field needs an alternative
	// testing strategy.
	t.Run("AttributeFilter", func(t *testing.T) {
		allowed := attribute.String("key", "val")
		filter := func(kv attribute.KeyValue) bool {
			return kv == allowed
		}
		mask := Stream{AttributeFilter: filter}
		got, match := NewView(completeIP, mask)(completeIP)
		require.True(t, match, "view did not match exact criteria")
		require.NotNil(t, got.AttributeFilter, "AttributeFilter not set")
		assert.True(t, got.AttributeFilter(allowed), "wrong AttributeFilter")
		other := attribute.String("key", "other val")
		assert.False(t, got.AttributeFilter(other), "wrong AttributeFilter")
	})
}

type badAgg struct {
	aggregation.Aggregation
	err error
}

func (a badAgg) Copy() aggregation.Aggregation { return a }

func (a badAgg) Err() error { return a.err }

func TestNewViewAggregationErrorLogged(t *testing.T) {
	tLog := testr.NewWithOptions(t, testr.Options{Verbosity: 6})
	l := &logCounter{LogSink: tLog.GetSink()}
	otel.SetLogger(logr.New(l))

	agg := badAgg{err: assert.AnError}
	mask := Stream{Aggregation: agg}
	got, match := NewView(completeIP, mask)(completeIP)
	require.True(t, match, "view did not match exact criteria")
	assert.Nil(t, got.Aggregation, "erroring aggregation used")
	assert.Equal(t, 1, l.ErrorN())
}
