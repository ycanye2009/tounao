package util

import (
	"log"
	"github.com/huichen/sego"
	"strings"
	"bytes"
	"os/exec"
	"net"
)

var segmenter sego.Segmenter

var Auto = false

func init() {
	segmenter.LoadDictionary("vendor/github.com/huichen/sego/data/dictionary.txt")
}

func Check(e error) (bool) {
	if e != nil {
		log.Fatalln(e)
		return false
	}
	return true
}

func Split(str string) []string {
	segments := segmenter.Segment([]byte(str))
	return sego.SegmentsToSlice(segments, true)
}

func RunWithAdb(args ...string) {

	if Auto {

		log.Printf("adb %s\n", strings.Join(args, " "))

		var buffer bytes.Buffer
		cmd := exec.Command("adb", args...)
		cmd.Stdout = &buffer
		cmd.Stderr = &buffer
		err := cmd.Run()
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		if err != nil {
			log.Printf("adb %s: %s\n", strings.Join(args, " "), err)
		}

	}
}

func HostIP() (ip string) {
	addrs, _ := net.InterfaceAddrs()

	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}

		}
	}
	return
}
