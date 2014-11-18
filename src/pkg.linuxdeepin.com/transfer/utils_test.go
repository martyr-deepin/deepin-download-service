package transfer

import (
	"os"
	"testing"
)

func TestCheckWritePermission(t *testing.T) {
	dir := "/tmp/testCheckWrtePermission/"
	os.Remove(dir)
	os.Mkdir(dir, 0755)
	if !CheckWritePermission(uint32(os.Getpid()), dir) {
		t.Errorf("Failed: %v write %v", os.Getpid(), dir)
	}

	os.Chmod(dir, 0000)
	if CheckWritePermission(uint32(os.Getpid()), dir) {
		t.Error("Failed")
	}

	//fix: root can write any
	if !CheckWritePermission(1, dir) {
		t.Error("Failed")
	}

	os.Chown(dir, 0, 0)
	if CheckWritePermission(uint32(os.Getpid()), dir) {
		t.Error("Failed")
	}

}
