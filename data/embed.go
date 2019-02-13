package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"time"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}
func main() {
	f, err := os.Create("embedded.go")
	check(err)
	defer f.Close()

	f.WriteString("package main\n\n")
	js, err := ioutil.ReadFile("data/sync.js")
	check(err)
	f.WriteString(fmt.Sprintf("const js = `%s`\n\n", js))
	f.WriteString(fmt.Sprintf(`const jsFile = "%s.js"`+"\n\n", RandStringBytesMaskImprSrc(10)))

	b, err := ioutil.ReadFile("data/index.html")
	check(err)
	f.WriteString(fmt.Sprintf("const defaultHTML = `%s`\n\n", b))

}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

func RandStringBytesMaskImprSrc(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
