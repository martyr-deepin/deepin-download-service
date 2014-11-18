package transfer

//#include <grp.h>
//#include <pwd.h>
import "C"

import (
	"fmt"
	"path/filepath"
	"syscall"
)

const (
	Kilo = 1024
	Mega = Kilo * Kilo
)

func PermissionVerfiy(pid uint32, fullname string) error {
	if CheckWritePermission(pid, filepath.Dir(fullname)) {
		return nil
	}
	// TODO: ask for root
	return fmt.Errorf("No Permission")
}

func GetGids(uid uint32, gid uint32) []uint32 {
	passwd := C.getpwuid(C.__uid_t(uid))
	grps := make([]C.__gid_t, 65535)
	n := (C.int)(65535)
	C.getgrouplist(passwd.pw_name, C.__gid_t(gid), (*C.__gid_t)(&grps[0]), &n)

	var gids []uint32
	for _, v := range grps {
		gids = append(gids, uint32(v))
	}
	return gids
}

func CheckWritePermission(pid uint32, dir string) bool {
	var pidStat syscall.Stat_t
	proc := fmt.Sprintf("/proc/%v", pid)
	syscall.Stat(proc, &pidStat)

	//if root, always can write
	if 0 == pidStat.Uid {
		return true
	}

	//verfiy pid can write dir
	var dirStat syscall.Stat_t
	syscall.Stat(dir, &dirStat)

	dirMode := dirStat.Mode
	if dirMode&0002 != 0 {
		//every one write
		return true
	}

	if dirMode&0020 != 0 {
		//Get Group of dir uid
		gids := GetGids(dirStat.Uid, dirStat.Gid)
		for _, v := range gids {
			if pidStat.Gid == v {
				return true
			}
		}
	}

	if dirMode&0200 != 0 {
		if pidStat.Uid == dirStat.Uid {
			return true
		}
	}

	return false
}
