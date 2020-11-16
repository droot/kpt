// Copyright 2020 Google LLC
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

package live

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/object"
)

// ResourceGroupGVK is the group/version/kind of the custom
// resource used to store inventory.
var ResourceGroupGVK = schema.GroupVersionKind{
	Group:   "kpt.dev",
	Version: "v1alpha1",
	Kind:    "ResourceGroup",
}

// InventoryResourceGroup wraps a ResourceGroup resource and implements
// the Inventory interface. This wrapper loads and stores the
// object metadata (inventory) to and from the wrapped ResourceGroup.
type InventoryResourceGroup struct {
	inv      *unstructured.Unstructured
	objMetas []object.ObjMetadata
}

var _ inventory.InventoryInfo = &InventoryResourceGroup{}

// WrapInventoryObj takes a passed ResourceGroup (as a resource.Info),
// wraps it with the InventoryResourceGroup and upcasts the wrapper as
// an the Inventory interface.
func WrapInventoryObj(obj *unstructured.Unstructured) inventory.Inventory {
	klog.V(4).Infof("wrapping inventory info")
	return &InventoryResourceGroup{inv: obj}
}

func WrapInventoryInfoObj(obj *unstructured.Unstructured) inventory.InventoryInfo {
	return &InventoryResourceGroup{inv: obj}
}

func InvToUnstructuredFunc(inv inventory.InventoryInfo) *unstructured.Unstructured {
	switch invInfo := inv.(type) {
	case *InventoryResourceGroup:
		return invInfo.inv
	default:
		return nil
	}
}

func (icm *InventoryResourceGroup) Name() string {
	return icm.inv.GetName()
}

func (icm *InventoryResourceGroup) Namespace() string {
	return icm.inv.GetNamespace()
}

func (icm *InventoryResourceGroup) ID() string {
	labels := icm.inv.GetLabels()
	if val, found := labels[common.InventoryLabel]; found {
		return val
	}
	return ""
}

// Load is an Inventory interface function returning the set of
// object metadata from the wrapped ResourceGroup, or an error.
func (icm *InventoryResourceGroup) Load() ([]object.ObjMetadata, error) {
	objs := []object.ObjMetadata{}
	if icm.inv == nil {
		return objs, fmt.Errorf("inventory info is nil")
	}
	klog.V(4).Infof("loading inventory...")
	items, exists, err := unstructured.NestedSlice(icm.inv.Object, "spec", "resources")
	if err != nil {
		err := fmt.Errorf("error retrieving object metadata from inventory object")
		return objs, err
	}
	if !exists {
		klog.V(4).Infof("Inventory (spec.resources) does not exist")
		return objs, nil
	}
	klog.V(4).Infof("loading %d inventory items", len(items))
	for _, itemUncast := range items {
		item := itemUncast.(map[string]interface{})
		namespace, _, err := unstructured.NestedString(item, "namespace")
		if err != nil {
			return []object.ObjMetadata{}, err
		}
		name, _, err := unstructured.NestedString(item, "name")
		if err != nil {
			return []object.ObjMetadata{}, err
		}
		group, _, err := unstructured.NestedString(item, "group")
		if err != nil {
			return []object.ObjMetadata{}, err
		}
		kind, _, err := unstructured.NestedString(item, "kind")
		if err != nil {
			return []object.ObjMetadata{}, err
		}
		groupKind := schema.GroupKind{
			Group: strings.TrimSpace(group),
			Kind:  strings.TrimSpace(kind),
		}
		klog.V(4).Infof("creating obj metadata: %s/%s/%s", namespace, name, groupKind)
		objMeta, err := object.CreateObjMetadata(namespace, name, groupKind)
		if err != nil {
			return []object.ObjMetadata{}, err
		}
		objs = append(objs, objMeta)
	}
	return objs, nil
}

// Store is an Inventory interface function implemented to store
// the object metadata in the wrapped ResourceGroup. Actual storing
// happens in "GetObject".
func (icm *InventoryResourceGroup) Store(objMetas []object.ObjMetadata) error {
	icm.objMetas = objMetas
	return nil
}

// GetObject returns the wrapped object (ResourceGroup) as a resource.Info
// or an error if one occurs.
func (icm *InventoryResourceGroup) GetObject() (*unstructured.Unstructured, error) {
	if icm.inv == nil {
		return nil, fmt.Errorf("inventory info is nil")
	}
	klog.V(4).Infof("getting inventory resource group")
	// Create a slice of Resources as empty Interface
	klog.V(4).Infof("Creating list of %d resources", len(icm.objMetas))
	var objs []interface{}
	for _, objMeta := range icm.objMetas {
		klog.V(4).Infof("storing inventory obj refercence: %s/%s", objMeta.Namespace, objMeta.Name)
		objs = append(objs, map[string]interface{}{
			"group":     objMeta.GroupKind.Group,
			"kind":      objMeta.GroupKind.Kind,
			"namespace": objMeta.Namespace,
			"name":      objMeta.Name,
		})
	}
	// Create the inventory object by copying the template.
	invCopy := icm.inv.DeepCopy()
	// Adds or clears the inventory ObjMetadata to the ResourceGroup "spec.resources" section
	if len(objs) == 0 {
		klog.V(4).Infoln("clearing inventory resources")
		unstructured.RemoveNestedField(invCopy.UnstructuredContent(),
			"spec", "resources")
	} else {
		klog.V(4).Infof("storing inventory (%d) resources", len(objs))
		err := unstructured.SetNestedSlice(invCopy.UnstructuredContent(),
			objs, "spec", "resources")
		if err != nil {
			return nil, err
		}
	}
	return invCopy, nil
}

// IsResourceGroupInventory returns true if the passed object is
// a ResourceGroup inventory object; false otherwise. If an error
// occurs, then false is returned and the error.
func IsResourceGroupInventory(obj *unstructured.Unstructured) (bool, error) {
	if obj == nil {
		return false, fmt.Errorf("inventory object is nil")
	}
	if !inventory.IsInventoryObject(obj) {
		return false, nil
	}
	invGK := obj.GetObjectKind().GroupVersionKind().GroupKind()
	if ResourceGroupGVK.GroupKind() != invGK {
		return false, nil
	}
	return true, nil
}
