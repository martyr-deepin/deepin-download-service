package transfer

import (
	"os"
)

const (
	TmpDir = "../../../misc/test/tmp"
)

func init() {
	os.Mkdir(TmpDir, 0755)
}
