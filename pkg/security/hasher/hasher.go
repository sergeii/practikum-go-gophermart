package hasher

type PasswordHasher interface {
	Hash(plain string) (string, error)
	Check(plain, hashed string) (bool, error)
}
