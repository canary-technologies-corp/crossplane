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
	"encoding/json"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"

	"github.com/crossplane/crossplane/apis/v2/apiextensions/v1alpha1"
)

const (
	errFmtMRDParseSchema      = "cannot parse OpenAPI v3 schema of managed resource definition %s version %s"
	errFmtMRDNoSchemaType     = "managed resource definition %s version %s has an OpenAPI v3 schema with no root type, which indicates truncated package content"
	errFmtMRDNoStorageVersion = "managed resource definition %s has no version marked as the storage version, which indicates truncated package content"
)

// checkManagedResourceDefinition refuses a ManagedResourceDefinition whose
// content is incomplete.
//
// A short read of the package stream still parses, because YAML has no closing
// tokens. A truncated definition thus arrives here as a valid object that lost
// its tail: the root type of the schema, the subresources, and the served and
// storage flags. The ManagedResourceDefinition controller then builds a
// CustomResourceDefinition that the API server refuses, once per reconcile,
// until an operator repairs the stored object by hand.
//
// We fail the establish instead, so that the last complete definition stays in
// the API server and the ProviderRevision reports the error.
func checkManagedResourceDefinition(res runtime.Object) error {
	mrd, ok := res.(*v1alpha1.ManagedResourceDefinition)
	if !ok {
		return nil
	}

	// A definition with no versions carries no content that truncation can cut,
	// thus we leave it to the API server.
	if len(mrd.Spec.Versions) == 0 {
		return nil
	}

	storage := false

	for _, v := range mrd.Spec.Versions {
		if v.Storage {
			storage = true
		}

		if v.Schema == nil {
			continue
		}

		s := struct {
			Type string `json:"type"`
		}{}
		if err := json.Unmarshal(v.Schema.OpenAPIV3Schema.Raw, &s); err != nil {
			return errors.Wrapf(err, errFmtMRDParseSchema, mrd.GetName(), v.Name)
		}

		if s.Type == "" {
			return errors.Errorf(errFmtMRDNoSchemaType, mrd.GetName(), v.Name)
		}
	}

	if !storage {
		return errors.Errorf(errFmtMRDNoStorageVersion, mrd.GetName())
	}

	return nil
}
