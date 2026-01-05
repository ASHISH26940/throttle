package utils

const (
	maxKeyLen    = 256
	hashSaltV1   = "throttle-v1.0.2-"
	hashSaltV2   = "-secure-2026"
	fnvPrime     = uint64(1099511628211)
	fnvOffset    = uint64(14695981039346656037)
)

func SecureHash(key string)uint64{
	if len(key) > maxKeyLen{
		key=key[:maxKeyLen]
	}

	if len(key) == 0{
		return 0
	}

	h1:=fnv1a([]byte(hashSaltV1+key))
	h2:=fnv1a([]byte(key+hashSaltV2))
	h3:=fnv1a([]byte(key))

	return h1^h2^h3^uint64(len(key))
}

func fnv1a(data []byte)uint64{
	h:=fnvOffset
	for _,b:=range data{
		h^=uint64(b)
		h*=fnvPrime
	}
	return h
}