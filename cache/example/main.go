package main

import (
	"fmt"
	"io/ioutil"
	"logauditer/cache"
	"os"
)

func main() {

	c := cache.NewCache()
	c.Set("a", "123")

	b, _ := c.Marshal()

	ioutil.WriteFile("data", b, 0777)

	c1 := cache.NewCache()

	ff, err := os.Open("data")
	if err != nil {
		//
	}

	c1.UnMarshal(ff)
	x, err := c1.Get("a")
	if err != nil {
		fmt.Fprintf(os.Stdout, "%v\n", err)
	}
	fmt.Fprintf(os.Stdout, "%v\n", x)
}
