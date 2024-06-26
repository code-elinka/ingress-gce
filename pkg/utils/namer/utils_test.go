/*
Copyright 2019 The Kubernetes Authors.
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

package namer

import (
	"fmt"
	"k8s.io/klog/v2"
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/ingress-gce/pkg/utils/common"
)

func TestTrimFieldsEvenly(t *testing.T) {
	t.Parallel()
	longString := "01234567890123456789012345678901234567890123456789"
	testCases := []struct {
		desc   string
		fields []string
		expect []string
		max    int
	}{
		{
			"no change",
			[]string{longString},
			[]string{longString},
			100,
		},
		{
			"equal to max and no change",
			[]string{longString, longString},
			[]string{longString, longString},
			100,
		},
		{
			"equally trimmed to half",
			[]string{longString, longString},
			[]string{longString[:25], longString[:25]},
			50,
		},
		{
			"trimmed to only 10",
			[]string{longString, longString, longString},
			[]string{longString[:4], longString[:3], longString[:3]},
			10,
		},
		{
			"trimmed to only 3",
			[]string{longString, longString, longString},
			[]string{longString[:1], longString[:1], longString[:1]},
			3,
		},
		{
			"one long field with one short field",
			[]string{longString, longString[:1]},
			[]string{longString[:1], ""},
			1,
		},
		{
			"one long field with one short field and trimmed to 5",
			[]string{longString, longString[:1]},
			[]string{longString[:5], ""},
			5,
		},
	}

	for _, tc := range testCases {
		res := TrimFieldsEvenly(tc.max, tc.fields...)
		if len(res) != len(tc.expect) {
			t.Fatalf("%s: expect length == %d, got %d", tc.desc, len(tc.expect), len(res))
		}

		totalLen := 0
		for i := range res {
			totalLen += len(res[i])
			if res[i] != tc.expect[i] {
				t.Errorf("%s: the %d field is want to be %q, but got %q", tc.desc, i, tc.expect[i], res[i])
			}
		}

		if tc.max < totalLen {
			t.Errorf("%s: expect totalLen to be less than %d, but got %d", tc.desc, tc.max, totalLen)
		}
	}
}

// TestFrontendNamingScheme asserts that correct naming scheme is returned for given ingress.
func TestFrontendNamingScheme(t *testing.T) {
	testCases := []struct {
		finalizer    string
		expectScheme Scheme
	}{
		{"", V1NamingScheme},
		{common.FinalizerKey, V1NamingScheme},
		{common.FinalizerKeyV2, V2NamingScheme},
	}
	for _, tc := range testCases {
		desc := fmt.Sprintf("Finalizer %q", tc.finalizer)
		t.Run(desc, func(t *testing.T) {
			ing := newIngress("namespace", "name")
			if tc.finalizer != "" {
				ing.ObjectMeta.Finalizers = []string{tc.finalizer}
			}

			if diff := cmp.Diff(tc.expectScheme, FrontendNamingScheme(ing, klog.TODO())); diff != "" {
				t.Fatalf("Got diff for Frontend naming scheme (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsValidGCEResourceName(t *testing.T) {
	for _, tc := range []struct {
		desc          string
		name          string
		expectIsValid bool
	}{
		{
			desc: "nil string",
		},
		{
			desc: "invalid name starts with numeric",
			name: "2testname",
		},
		{
			desc: "invalid name with dot character",
			name: "test.name",
		},
		{
			desc: "invalid name with capitals",
			name: "testName",
		},
		{
			desc: "invalid name with trailing -",
			name: "test-name-",
		},
		{
			desc: "invalid name with all numerics",
			name: "123243",
		},
		{
			desc:          "valid name",
			name:          "test-name-123243",
			expectIsValid: true,
		},
	} {
		if got := isValidGCEResourceName(tc.name); got != tc.expectIsValid {
			t.Errorf("isValidGCEResourceName(%s) = %t, want %t", tc.name, got, tc.expectIsValid)
		}
	}
}
