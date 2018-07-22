// Package rfsb provides the primitives to build configuration management systems.
//
// A Basic Example
//
// As a starting point, a very basic usage example:
//
//    package main
//
//    import (
//        "os"
//
//        "github.com/sirupsen/logrus"
//        "github.com/lclarkmichalek/rfsb"
//    )
//
//    func main() {
//        rfsb.SetupDefaultLogging()
//
//        exampleFile := &rfsb.FileResource{
//            Path: "/tmp/example",
//            Contents: "bar",
//            Mode: 0644,
//            UID: uint32(os.Getuid()),
//            GID: uint32(os.Getgid()),
//        }
//        rfsb.Register("exampleFile", exampleFile)
//
//        err := rfsb.Materialize(context.Background())
//        if err != nil {
//            logrus.Fatalf("materialization failed: %v", err)
//    	}
//    }
//
// This is a complete rfsb config management system that defines a resource to be created, and then creates it
// (materializes it).
//
// Resources
//
// The atomic units of computation in the RFSB model are types implementing the Resource interface. This is a pretty
// simple interface, basically being 'a thing that can be run (and has a name, and a logger)'. RFSB comes with most of
// the resources a standard configuration management system would need, but if other resources are required, the
// interface is simple to understand and implement.
//
// Resources can be composed together via a ResourceGraph:
//
//   func defineExampleFiles() Resource {
//        exampleFile := &rfsb.FileResource{
//            Path: "/tmp/example",
//            Contents: "bar",
//            Mode: 0644,
//            UID: uint32(os.Getuid()),
//            GID: uint32(os.Getgid()),
//        }
//        secondExampleFile := &rfsb.FileResource{
//            Path: "/tmp/example2",
//            Contents: "boz",
//            Mode: 0644,
//            UID: uint32(os.Getuid()),
//            GID: uint32(os.Getgid()),
//        }
//        rg := &rfsb.ResourceGraph{}
//        rg.Register("example", exampleFile)
//        rg.Register("example2", secondExampleFile)
//        return rg
//    }
//
// Which can be then composed further, building up higher level abstractions:
//
//   func provisionBaseFilesystem() Resource {
//        rg := &rfsb.ResourceGraph{}
//        files := defineExampleFiles()
//        rg.Register("exampleFiles", files)
//        sshd := &rfsb.FileResource{
//            Path: "/etc/ssh/sshd_config",
//            Mode: 0600,
//            UID: 0,
//            GID: 0,
//            Contents: `UseDNS no`,
//        }
//        rg.Register("sshd", sshd)
//        return rg
//   }
//
// Resources (and ResourceGraphs) can also have dependencies between them:
//
//   func provisionAdminUsers() Resource {...}
//
//   func provision() Resource {
//        rg := &rfsb.ResourceGraph{}
//        filesystem := provisionBaseFilesystem()
//        rg.Register("filesystem", filesystem)
//        users := provisionAdminUsers()
//        rg.When(filesystem).Do(users)
//        return rg
//   }
//
// The `When` and `Do` methods here set up a Dependency between the filesystem provisioning resource and the admin
// users provisioning resource. This ensures that admin users will not be provisioned before the filesystem has been
// set up.
//
// When all of the Resources have been defined, and composed together into a single ResourceGraph, we can Materialize
// the graph:
//
//    func main() {
//        root := provision()
//        err := root.Materialize(context.Background())
//        if err != nil {
//            os.Stderr.Write(err.Error())
//            os.Exit(1)
//        }
//    }
//
// And there we have it. Everything you need to know about rfsb.
package rfsb
