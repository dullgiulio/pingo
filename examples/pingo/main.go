package main

import (
	"fmt"
	"github.com/dullgiulio/pingo"
)

func runPlugin(proto, path string) {
	p := pingo.NewPlugin(proto, path)
	p.Start()
	defer p.Stop()

	objs, err := p.Objects()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Objects: %s\n", objs)

	var resp string

	if err := p.Call("Plugin.SayHello", "from your plugin", &resp); err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("%s\n", resp)
	}
	if err := p.Call("Plugin.SayHello", "from your plugin, second call", &resp); err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("%s\n", resp)
	}
}

func main() {
	protocols := []string{"unix", "tcp"}
	for _, p := range protocols {
		fmt.Printf("Running hello world plugin via %s\n", p)

		runPlugin(p, "bin/plugins/pingo-hello-world")

		fmt.Println("Plugin terminated.")
	}

	fmt.Println("Running plugin that fails to register in time")

	runPlugin("tcp", "bin/plugins/pingo-sleep")

	fmt.Println("Plugin terminated.")
}
