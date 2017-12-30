/*
 * Copyright (C) 2017-2018 GIG Technology NV and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package itsyouonline

// Permission defines itsyouonline permission for an org & namespace
type Permission struct {
	Read   bool
	Write  bool
	Delete bool
	Admin  bool
}

func (p Permission) perms() []string {
	var perms []string
	if p.Read {
		perms = append(perms, "read")
	}
	if p.Write {
		perms = append(perms, "write")
	}
	if p.Delete {
		perms = append(perms, "delete")
	}
	if p.Admin {
		perms = append(perms, "admin")
	}
	return perms
}

// Scopes returns the scopes this permission structure should grant you,
// within the context of the given organization and namespace.
func (p Permission) Scopes(org, namespace string) []string {
	var scopes []string
	scopePrefix := "user:memberof:" + org + "." + namespace + "."
	for _, p := range p.perms() {
		if p == "admin" {
			scopes = append(scopes, scopePrefix[:len(scopePrefix)-1])
		} else {
			scopes = append(scopes, scopePrefix+p)
		}
	}
	return scopes
}
