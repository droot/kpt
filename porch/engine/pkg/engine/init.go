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

package engine

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/pkg/kptpkg"
	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/engine/pkg/kpt"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

type initPackageMutation struct {
	name string
	spec api.PackageInitTaskSpec
}

var _ mutation = &initPackageMutation{}

func (m *initPackageMutation) Apply(ctx context.Context, resources repository.PackageResources) (repository.PackageResources, *api.Task, error) {

	fs := filesys.MakeFsInMemory()
	pkgPath := filepath.Join("/", m.name)
	if err := fs.Mkdir(pkgPath); err != nil {
		return repository.PackageResources{}, nil, err
	}
	initializer := kpt.NewInitializer()

	err := initializer.Initialize(ctx, fs, kptpkg.InitOptions{
		PkgPath:  pkgPath,
		Desc:     m.spec.Description,
		Keywords: m.spec.Keywords,
		Site:     m.spec.Site,
	})
	if err != nil {
		return repository.PackageResources{}, nil, fmt.Errorf("failed to initialize pkg %q: %w", m.name, err)
	}

	result, err := readResources(fs)
	if err != nil {
		return repository.PackageResources{}, nil, err
	}

	return result, &api.Task{}, nil
}
