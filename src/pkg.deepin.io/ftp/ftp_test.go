package ftp

import (
	//	"fmt"
	"io/ioutil"
	"pkg.deepin.io/lib/utils"
	"testing"
)

func TestFptDownload(t *testing.T) {
	c, err := Connect("localhost:8021")
	if err != nil {
		// TODO: Howerver, if can not connect to server, stop testing and waring.
		t.Log("WARNING: Cannot Connect to ftp server, please check if fpt server active at localhost:8021.")
		return
	}
	err = c.Login("anonymous", "anonymous")
	if err != nil {
		t.Fatal(err)
	}
	err = c.NoOp()
	if err != nil {
		t.Error(err)
	}

	//	size, err := c.Size("ubuntu/pool/universe/a/audacious/libaudclient2_3.4.3-1_amd64.deb")
	//	fmt.Println("Size: ", size, " Error: ", err)

	r, err := c.Retr("public/test")
	if err != nil {
		t.Error(err)
	} else {
		buf, err := ioutil.ReadAll(r)
		if err != nil {
			t.Error(err)
		}
		md5Str, ok := utils.SumStrMd5(string(buf))
		if !ok {
			t.Error("Check Md5Sum Failed")
		}
		t.Log(md5Str)
		if "77f1a79cd3b26c493eeadf834038feb0" != md5Str {
			t.Error("Check Md5Sum Failed")
		}
		r.Close()
	}
}
