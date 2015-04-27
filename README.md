# Pingo: Plugins for Go

Pingo is a simple standalone library to create plugins for your Go program. As Go is statically
linked, all plugins run as external processes.

The library aims to be as simple as possible and to mimic the standard RPC package to be
immediately familiar to most developers.

Pingo supports both TCP and Unix as communication protocols. However, remote plugins are currently
not supported.  Remote plugins might be implemented if requested.

## Example

Create a new plugin. Make a directory named after the plugin (for example "plugins/hello-world")
and write "main.go" as follows:

```go
// Always create a new binary
package main

import "github.com/dullgiulio/pingo"

// Create an object to be exported
type MyPlugin struct{}

// Exported method, with a RPC signature
func (p *MyPlugin) SayHello(name string, msg *string) error {
    *msg = "Hello, " + name
    return nil
}

func main() {
	plugin := &MyPlugin{}

	// Register the objects to be exported
	pingo.Register(plugin)
	// Run the main events handler
	pingo.Run()
}
```

And compile it:
```sh
$ cd plugins/hello-world
$ go build
```

You should get an executable called "hello-world". Congratulations, this is your plugin.

Now, time to use the newly create plugin.

In your main executable, invoke the plugin you have just created:

```go
package main

import (
	"log"
	"github.com/dullgiulio/pingo"
)

func main() {
	// Make a new plugin from the executable we created. Connect to it via TCP
	p := pingo.NewPlugin("tcp", "plugins/hello-world/hello-world")
	// Actually start the plugin
	p.Start()
	// Remember to stop the plugin when done using it
	defer p.Stop()

	var resp string

	// Call a function from the object we created previously
	if err := p.Call("MyPlugin.SayHello", "Go developer", &resp); err != nil {
		log.Print(err)
	} else {
		log.Print(resp)
	}
}
```

Now, build your executable and all should work!  Remember to use the correct path to
your plugins when you make the Plugin object.  Ideally, always pass an absolute path.

## Bugs

Report bugs in Github.  Pull requests are welcome!

## License

MIT
