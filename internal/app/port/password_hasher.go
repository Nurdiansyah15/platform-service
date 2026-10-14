package port

// PasswordHasher meng-hash dan memverifikasi password user.
type PasswordHasher interface {
	Hash(password string) (string, error)
	// Verify mengembalikan true bila password cocok dengan hash.
	Verify(hash, password string) bool
}
