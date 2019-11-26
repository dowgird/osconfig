//  Copyright 2019 Google Inc. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.
//
// Package instance implements VM instance software configs.
package instance

import (
	"bytes"
	"encoding/json"

	agentendpointpb "github.com/GoogleCloudPlatform/osconfig/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/agentendpoint/v1alpha1"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

// LocalConfig represents the structure of the config to the JSON parser.
//
// The types of members of the struct are wrappers for protobufs and delegate
// the parsing to jsonpb lib via their UnmarshalJSON implementations.
type LocalConfig struct {
	Packages            []Package
	PackageRepositories []PackageRepository
	SoftwareRecipes     []SoftwareRecipe
}

type Package struct {
	i agentendpointpb.Package
}

func (r *Package) UnmarshalJSON(b []byte) error {
	rd := bytes.NewReader(b)
	return jsonpb.Unmarshal(rd, &r.i)
}

type PackageRepository struct {
	i agentendpointpb.PackageRepository
}

func (r *PackageRepository) UnmarshalJSON(b []byte) error {
	rd := bytes.NewReader(b)
	return jsonpb.Unmarshal(rd, &r.i)
}

type SoftwareRecipe struct {
	r agentendpointpb.SoftwareRecipe
}

func (r *SoftwareRecipe) UnmarshalJSON(b []byte) error {
	rd := bytes.NewReader(b)
	return jsonpb.Unmarshal(rd, &r.r)
}

func ParseLocalConfig(a []byte) (rv LocalConfig, e error) {
	e = json.Unmarshal(a, &rv)
	return
}

// GetId returns a repository Id that is used to group repositories for
// override by higher priotiry policy(-ies).
// For repositories that have no such Id, GetId returns nil, in which
// case the repository is never overridden.
func GetId(repo agentendpointpb.PackageRepository) *string {
	switch repo.Repository.(type) {
	case *agentendpointpb.PackageRepository_Yum:
		id := "yum-" + repo.GetYum().GetId()
		return &id
	case *agentendpointpb.PackageRepository_Zypper:
		id := "zypper-" + repo.GetZypper().GetId()
		return &id
	default:
		return nil

	}
}

// MergeConfigs merges the local config with the lookup response, giving priority to the global
// response.
func MergeConfigs(local LocalConfig, global agentendpointpb.LookupEffectiveGuestPoliciesResponse) (r agentendpointpb.LookupEffectiveGuestPoliciesResponse) {
	// Ids that are in the maps below
	repos := make(map[string]bool)
	pkgs := make(map[string]bool)
	recipes := make(map[string]bool)

	for _, v := range global.GetPackages() {
		pkgs[v.Package.Name] = true
		r.Packages = append(r.Packages, v)
	}
	for _, v := range global.GetPackageRepositories() {
		if id := GetId(*v.PackageRepository); id != nil {
			repos[*id] = true
		}
		r.PackageRepositories = append(r.PackageRepositories, v)
	}
	for _, v := range global.GetSoftwareRecipes() {
		recipes[v.SoftwareRecipe.Name] = true
		r.SoftwareRecipes = append(r.SoftwareRecipes, v)
	}

	for _, v := range local.Packages {
		if _, ok := pkgs[v.i.Name]; !ok {
			sp := new(agentendpointpb.LookupEffectiveGuestPoliciesResponse_SourcedPackage)
			sp.Package = &v.i
			r.Packages = append(r.Packages, sp)
		}
	}
	for _, v := range local.PackageRepositories {
		id := GetId(v.i)
		if id != nil {
			if _, ok := repos[*id]; ok {
				continue
			}
		}
		sr := new(agentendpointpb.LookupEffectiveGuestPoliciesResponse_SourcedPackageRepository)
		sr.PackageRepository = &v.i
		r.PackageRepositories = append(r.PackageRepositories, sr)

	}
	for _, v := range local.SoftwareRecipes {
		if _, ok := recipes[v.r.Name]; !ok {
			sp := new(agentendpointpb.LookupEffectiveGuestPoliciesResponse_SourcedSoftwareRecipe)
			sp.SoftwareRecipe = proto.Clone(&v.r).(*agentendpointpb.SoftwareRecipe)
			r.SoftwareRecipes = append(r.SoftwareRecipes, sp)
		}

	}

	return
}
