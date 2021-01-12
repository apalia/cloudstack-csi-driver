package util

// RoundUpBytesToGB converts a size given in bytes to GB with
// an upper rounding (it gives the smallest amount in GB
// which is greater than the original amount)
func RoundUpBytesToGB(n int64) int64 {
	return (((n+1023)/1024+1023)/1024 + 1023) / 1024
}

// GigaBytesToBytes gives an exact conversion from GigaBytes to Bytes
func GigaBytesToBytes(gb int64) int64 {
	return gb * 1024 * 1024 * 1024
}
