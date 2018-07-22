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
//
// Rationale
//
// Puppet/Chef/Ansible (henceforth referred to collectively as CAP) all suck. For small deployments, the centralised
// server of CP is annoying, and for larger deployments, A is limiting, P devolves into a mess, and C, well I guess it
// can work if you devote an entire team to it. RFSB aims to grow gracefully from small to large, and give you the
// power to run it in any way you like. Let's take a look at some features that RFSB doesn't force you to provide.
//
// A popular feature of many systems is periodic agent runs. RFSB doesn't have any built in support for running
// periodically, so we'll need to build it ourselves. Let's take a look at what that might look like:
//
//    func main() {
//        root := provision()
//        for range time.NewTicker(time.Minute * 15) {
//            err := root.Materialize(context.Background())
//            if err != nil {
//                os.Stderr.Write(err.Error())
//            }
//        }
//    }
//
// Here we've used the radical method of running our Materialize function in a loop. But what you don't see is all the
// complexity you just avoided. For example, Chef supports both 'interactive' and 'daemon' mode. You can trigger an
// interactive Chef run via "chefctl -i". I think this communicates with the "chef-client" process, which then actually
// performs the Chef run, and reports the results back to chefctl. I think. In RFSB, this complexity goes away. If I
// want to understand how your RFSB job runs, I open its main.go and read. If you don't ever use periodic runs, you
// don't ever need to think about them.
//
// Another common feature is some kind of fact store. This allows you to see in one place all of the information about
// all of your servers. Or at least, all of the information that Puppet or Chef figured might be useful. In RFSB, we
// take the radical approach of letting you use a fact store if you want, or not if you don't want. For example, if you
// have a facts store with some Go bindings, you might use it to customize how you provision things:
//
//   func provision() Resource {
//        rg := &rfsb.ResourceGraph{}
//        filesystem := provisionBaseFilesystem()
//        rg.Register("filesystem", filesystem)
//
//        hostname, _ := os.Hostname()
//        role := myFactsStore.GetRoleFromHostname(hostname)
//        if role == "database" {
//            rg.When(filesystem).Do("database", provisionDatabase())
//        } else if role == "frontend" {
//            rg.When(filesystem).Do("frontend", provisionFrontend())
//        }
//        return rg
//   }
//
// Now, this might be the point where you cry out and say 'but I can't have a big if else statement for all of my
// roles!'. The reality is that you probably can, and if you can't you should cut down the number of unique
// configurations. But anyway, assuming you have a good reason, RFSB has ways to handle this. Mostly, it relies on an
// interesting property, known as 'being just Go'. For example, you could use an interesting abstraction known as the
// 'map':
//
//   var provisionRole := map[string]func() Resource{}{
//       "database": provisionDatabase(),
//       "frontend": provisionFrontend(),
//   }
//
//   func provision() Resource {
//        rg := &rfsb.ResourceGraph{}
//        filesystem := provisionBaseFilesystem()
//        rg.Register("filesystem", filesystem)
//
//        hostname, _ := os.Hostname()
//        role := myFactsStore.GetRoleFromHostname(hostname)
//        rg.When(filesystem).Do(role, provisionRole[role]())
//        return rg
//   }
//
// Now you might say "but that's basically just a switch statement in map form". And you'd be correct. But you get the
// point. You have all the power of Go at your disposal. Go build the abstractions that make sense to you. I'm not
// going to tell you how to build your software; the person with the context needed to solve your problems is you.
package rfsb
