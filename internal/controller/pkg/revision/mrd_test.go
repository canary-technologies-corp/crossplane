/*
Copyright 2026 The Crossplane Authors.

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

package revision

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/crossplane/crossplane/apis/v2/apiextensions/v1alpha1"
)

func mrd(name string, versions ...v1alpha1.CustomResourceDefinitionVersion) *v1alpha1.ManagedResourceDefinition {
	return &v1alpha1.ManagedResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: v1alpha1.ManagedResourceDefinitionSpec{
			CustomResourceDefinitionSpec: v1alpha1.CustomResourceDefinitionSpec{
				Versions: versions,
			},
		},
	}
}

func mrdVersion(name string, storage bool, schema string) v1alpha1.CustomResourceDefinitionVersion {
	v := v1alpha1.CustomResourceDefinitionVersion{
		Name:    name,
		Served:  storage,
		Storage: storage,
	}
	if schema != "" {
		v.Schema = &v1alpha1.CustomResourceValidation{
			OpenAPIV3Schema: runtime.RawExtension{Raw: []byte(schema)},
		}
	}

	return v
}

func TestCheckManagedResourceDefinition(t *testing.T) {
	// The tail of this schema is cut, as a short read of the package stream
	// cuts it. The remaining part is still valid JSON, but the root type and
	// the storage flag are gone.
	truncated := `{"description":"Grant is the Schema for the Grants API.","properties":{"spec":{"properties":{"forProvider":{"type":"object"}}}}}`
	complete := `{"description":"Grant is the Schema for the Grants API.","properties":{"spec":{"type":"object"}},"type":"object"}`

	cases := map[string]struct {
		reason string
		obj    runtime.Object
		wantOK bool
	}{
		"NotAManagedResourceDefinition": {
			reason: "Objects of other kinds must pass without a check.",
			obj:    &extv1.CustomResourceDefinition{ObjectMeta: metav1.ObjectMeta{Name: "grants.postgresql"}},
			wantOK: true,
		},
		"Complete": {
			reason: "A definition with a storage version and a root type must pass.",
			obj:    mrd("grants.postgresql", mrdVersion("v1alpha1", true, complete)),
			wantOK: true,
		},
		"CompleteWithTwoVersions": {
			reason: "Only one version must be the storage version.",
			obj: mrd("grants.postgresql",
				mrdVersion("v1alpha1", false, complete),
				mrdVersion("v1beta1", true, complete)),
			wantOK: true,
		},
		"NoSchema": {
			reason: "A version without a schema must pass, because there is nothing to truncate.",
			obj:    mrd("grants.postgresql", mrdVersion("v1alpha1", true, "")),
			wantOK: true,
		},
		"TruncatedSchema": {
			reason: "A schema with no root type must fail, because the content is incomplete.",
			obj:    mrd("grants.postgresql", mrdVersion("v1alpha1", true, truncated)),
			wantOK: false,
		},
		"NoStorageVersion": {
			reason: "A definition with no storage version must fail, because the content is incomplete.",
			obj:    mrd("grants.postgresql", mrdVersion("v1alpha1", false, complete)),
			wantOK: false,
		},
		"UnparsableSchema": {
			reason: "A schema that is not valid JSON must fail.",
			obj:    mrd("grants.postgresql", mrdVersion("v1alpha1", true, `{"type":`)),
			wantOK: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := checkManagedResourceDefinition(tc.obj)
			if diff := cmp.Diff(tc.wantOK, err == nil); diff != "" {
				t.Errorf("\n%s\ncheckManagedResourceDefinition(...): -wantOK, +gotOK:\n%s\nerr: %v", tc.reason, diff, err)
			}
		})
	}
}
