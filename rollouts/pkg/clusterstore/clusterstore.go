// Copyright 2022 Google LLC
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

package clusterstore

import (
	"context"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"

	"golang.org/x/oauth2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	gkeclusterapis "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/container/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// ClusterStore represents a store of kubernetes cluster.
type ClusterStore struct {
	// Config/Client points to the config
	// pointing to where the rollout controller is running
	Config *rest.Config
	client.Client
	WorkloadIdentityHelper
}

func (cs *ClusterStore) Init() error {
	if err := cs.WorkloadIdentityHelper.Init(cs.Config); err != nil {
		return err
	}
	return nil
}

func (cs *ClusterStore) ListClusters(ctx context.Context, selectors ...*metav1.LabelSelector) (*gkeclusterapis.ContainerClusterList, error) {
	gkeClusters, err := cs.listClusters(ctx, selectors[0])
	if err != nil {
		return nil, err
	}

	for _, selector := range selectors[1:] {
		selectorClusters, err := cs.listClusters(ctx, selector)
		if err != nil {
			return nil, err
		}

		intersection := []gkeclusterapis.ContainerCluster{}

		for _, cluster := range gkeClusters.Items {
			for _, selectorCluster := range selectorClusters.Items {
				if cluster.Name == selectorCluster.Name {
					intersection = append(intersection, cluster)
					break
				}
			}
		}

		gkeClusters.Items = intersection
	}

	sort.Slice(gkeClusters.Items, func(i, j int) bool {
		return strings.Compare(gkeClusters.Items[i].Name, gkeClusters.Items[j].Name) == -1
	})

	return gkeClusters, nil
}

func (cs *ClusterStore) PrintClusterInfos(ctx context.Context, clusters *gkeclusterapis.ContainerClusterList) {
	logger := log.FromContext(ctx)
	for _, gkeCluster := range clusters.Items {
		logger.Info("gke clusters", "namespace", gkeCluster.Namespace, "name", gkeCluster.Name)
		for _, cond := range gkeCluster.Status.Conditions {
			logger.Info("gke cluster", "name", gkeCluster.Name, "condition", cond)
		}
	}
}

func (cs *ClusterStore) GetCluster(ctx context.Context, name string) (*gkeclusterapis.ContainerCluster, error) {
	gkeCluster := gkeclusterapis.ContainerCluster{}
	clusterKey := client.ObjectKey{
		Namespace: "config-control",
		Name:      name,
	}
	if err := cs.Get(ctx, clusterKey, &gkeCluster); err != nil {
		return nil, err
	}

	return &gkeCluster, nil
}

func (cs *ClusterStore) GetClusterClient(ctx context.Context, cluster *gkeclusterapis.ContainerCluster) (client.Client, dynamic.Interface, error) {
	clusterClientConfig, err := cs.getRESTConfig(ctx, cluster)
	if err != nil {
		return nil, nil, err
	}
	cl, err := client.New(clusterClientConfig, client.Options{})
	if err != nil {
		return nil, nil, err
	}
	dynCl, err := dynamic.NewForConfig(clusterClientConfig)
	if err != nil {
		return nil, nil, err
	}
	return cl, dynCl, err
}

func (cs *ClusterStore) listClusters(ctx context.Context, selector *metav1.LabelSelector) (*gkeclusterapis.ContainerClusterList, error) {
	gkeClusters := &gkeclusterapis.ContainerClusterList{}

	var opts []client.ListOption

	if selector != nil {
		selector, err := metav1.LabelSelectorAsSelector(selector)
		if err != nil {
			return nil, err
		}
		opts = append(opts, client.MatchingLabelsSelector{Selector: selector})
	}

	// TODO: make it configurable ?
	opts = append(opts, client.InNamespace("config-control"))
	if err := cs.List(ctx, gkeClusters, opts...); err != nil {
		return nil, err
	}

	return gkeClusters, nil
}

func (cs *ClusterStore) getRESTConfig(ctx context.Context, cluster *gkeclusterapis.ContainerCluster) (*rest.Config, error) {
	logger := log.FromContext(ctx)
	restConfig := &rest.Config{}
	clusterCaCertificate := cluster.Spec.MasterAuth.ClusterCaCertificate
	if clusterCaCertificate == nil || *clusterCaCertificate == "" {
		return nil, fmt.Errorf("cluster CA certificate data is missing")
	}
	caData, err := base64.StdEncoding.DecodeString(*clusterCaCertificate)
	if err != nil {
		return nil, fmt.Errorf("error decoding ca certificate: %w", err)
	}
	restConfig.CAData = caData
	if cluster.Status.Endpoint == "" {
		return nil, fmt.Errorf("cluster master endpoint field is empty")
	}
	restConfig.Host = "https://" + cluster.Status.Endpoint
	logger.Info("Host endpoint is", "endpoint", restConfig.Host)
	tokenSource, err := cs.getConfigConnectorContextTokenSource(ctx, cluster.GetNamespace())
	if err != nil {
		return nil, err
	}
	token, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("error getting token: %w", err)
	}
	restConfig.BearerToken = token.AccessToken
	return restConfig, nil
}

// getConfigConnectorContextTokenSource gets and returns the ConfigConnectorContext for the given namespace.
func (cs *ClusterStore) getConfigConnectorContextTokenSource(ctx context.Context, ns string) (oauth2.TokenSource, error) {
	// TODO: migrate to it's own Go type and use client.Client instance for it
	gvr := schema.GroupVersionResource{
		Group:    "core.cnrm.cloud.google.com",
		Version:  "v1beta1",
		Resource: "configconnectorcontexts",
	}

	cr, err := cs.dynamicClient.Resource(gvr).Namespace(ns).Get(ctx, "configconnectorcontext.core.cnrm.cloud.google.com", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	googleServiceAccount, _, err := unstructured.NestedString(cr.Object, "spec", "googleServiceAccount")
	if err != nil {
		return nil, fmt.Errorf("error reading spec.googleServiceAccount from ConfigConnectorContext in %q: %w", ns, err)
	}

	if googleServiceAccount == "" {
		return nil, fmt.Errorf("could not find spec.googleServiceAccount from ConfigConnectorContext in %q: %w", ns, err)
	}

	kubeServiceAccount := types.NamespacedName{
		Namespace: "cnrm-system",
		Name:      "cnrm-controller-manager-" + ns,
	}
	return cs.WorkloadIdentityHelper.GetGcloudAccessTokenSource(ctx, kubeServiceAccount, googleServiceAccount)
}
