package common

// todo @0xjackalope
func Encrypt(data interface{}, pubKey interface{}) interface{} {
	return nil
}
func Decrypt(data interface{}, privateKey interface{}) interface{} {
	return nil
}

func ByteEqual(a []byte, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
