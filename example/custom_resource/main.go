// An example showing how to set up a custom resource. When run, this file should print:
//
// INFO[0000] evaluating resource                           resource=foo
// INFO[0000] materializing resource                        resource=foo
// INFO[0000] hello world                                   resource=foo
// INFO[0000] resource evaluated in 550.401µs               resource=foo
package main

import (
	"context"

	"github.com/lclarkmichalek/rfsb"
	"github.com/sirupsen/logrus"
)

type MyCustomResource struct {
	rfsb.ResourceMeta

	Message string
}

func (mcr *MyCustomResource) Materialize(context.Context, chan<- rfsb.Signal) error {
	mcr.Logger().Infoln(mcr.Message)
	return nil
}

func main() {
	rfsb.SetupDefaultLogging()

	rg := &rfsb.ResourceGraph{}
	rg.Register("foo", &MyCustomResource{Message: "hello world"})

	err := rg.Materialize(context.Background(), make(chan rfsb.Signal, 1024))
	if err != nil {
		logrus.Fatal(err.Error())
	}
}
