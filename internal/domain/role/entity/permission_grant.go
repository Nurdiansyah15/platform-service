package entity

// ScopeCompany — scope_type permission role yang berlaku di seluruh company (tanpa nilai).
const ScopeCompany = "company"

// PermissionGrant — satu permission pada role beserta scope_type tempat ia berlaku.
// Nilai scope (branch mana, department mana) disimpan di role assignment.
type PermissionGrant struct {
	PermissionID string
	Code         string
	ScopeType    string
}

// PermissionInput — permission yang diminta dipasang ke role (scope_type kosong = company).
type PermissionInput struct {
	Code      string
	ScopeType string
}
