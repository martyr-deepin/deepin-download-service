package ftp

import (
	"fmt"
	"io/ioutil"
	"testing"
)

func Test_testApt(t *testing.T) {

	//	ftp://ftp.sjtu.edu.cn/ubuntu/pool/universe/a/audacious/libaudclient2_3.4.3-1_amd64.deb
	c, err := Connect("ftp.sjtu.edu.cn:21")
	if err != nil {
		t.Fatal(err)
	}
	err = c.Login("anonymous", "anonymous")
	if err != nil {
		t.Fatal(err)
	}
	err = c.NoOp()
	if err != nil {
		t.Error(err)
	}

	size, err := c.Size("ubuntu/pool/universe/a/audacious/libaudclient2_3.4.3-1_amd64.deb")
	fmt.Println("Size: ", size, " Error: ", err)

	r, err := c.Retr("ubuntu/pool/universe/a/audacious/libaudclient2_3.4.3-1_amd64.deb")
	if err != nil {
		t.Error(err)
	} else {
		_, err := ioutil.ReadAll(r)
		if err != nil {
			t.Error(err)
		}
		r.Close()
	}
}
