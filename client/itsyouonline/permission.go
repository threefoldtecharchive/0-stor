package itsyouonline

// Permission defines itsyouonline permission for an org & namespace
type Permission struct {
	Read   bool
	Write  bool
	Delete bool
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
	return perms
}

func (p Permission) scopes(org, namespace string) []string {
	var scopes []string
	scopePrefix := "user:memberof:" + org + "." + namespace + "."
	for _, p := range p.perms() {
		scopes = append(scopes, scopePrefix+p)
	}
	return scopes
}
