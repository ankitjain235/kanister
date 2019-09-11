// Copyright 2019 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package discovery

import (
	"context"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

func AllGVRs(ctx context.Context, cli discovery.DiscoveryInterface) ([]schema.GroupVersionResource, error) {
	arls, err := cli.ServerPreferredResources()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list APIResources")
	}
	return apiToGroupVersion(arls)
}

func NamespacedGVRs(ctx context.Context, cli discovery.DiscoveryInterface) ([]schema.GroupVersionResource, error) {
	arls, err := cli.ServerPreferredNamespacedResources()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list APIResources")
	}
	return apiToGroupVersion(arls)
}

func apiToGroupVersion(arls []*metav1.APIResourceList) ([]schema.GroupVersionResource, error) {
	gvrs := make([]schema.GroupVersionResource, 0, len(arls))
	for _, arl := range arls {
		gv, err := schema.ParseGroupVersion(arl.GroupVersion)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to Parse GroupVersion %s", arl.GroupVersion)
		}
		for _, ar := range arl.APIResources {
			// Although APIResources have Group and Version fields they're empty as of client-go v1.13.1
			gvrs = append(gvrs, schema.GroupVersionResource{
				Group:    gv.Group,
				Version:  gv.Version,
				Resource: ar.Name,
			})
		}
	}
	return gvrs, nil
}