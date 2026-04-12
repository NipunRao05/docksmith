package utils

import (
	"io"
)

func CopyFile(src io.Reader, dst io.Writer) (int64, error) {
	return io.Copy(dst, src)
}
