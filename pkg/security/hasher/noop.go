package hasher

type NoopPasswordHasher struct{}

func NewNoopPasswordHasher() NoopPasswordHasher {
	return NoopPasswordHasher{}
}

func (h NoopPasswordHasher) Hash(password string) (string, error) {
	return password, nil
}

func (h NoopPasswordHasher) Check(password1, password2 string) (bool, error) {
	return password1 == password2, nil
}
