# Really Foolish Server Bootstrapping

> Because sometimes you don't want to reprovision every machine in the fleet to add a sysctl

The workflow of Ansible meets the feature set of cloud-init plus the ease of deployment of bash script.


## Why: How do I deploy applications without tearing my eyes out

Let's compare deployment technologies on the market today:

| Name | Do you really like it? | Is it really wicked? |
| - | - | - |
| Kube | We're loving it like this | We're loving it like that |
| everything else | Doesn't even know what it takes to be a garage MC | Can't see any hands in the air |

Unfortunately, none of these quite hit the mark of "we're loving it loving it loving it". To hit the triple L, we need to break it down:

### L - Loving it

I like my machines. Sometimes I **log** onto them. When I do, I like stuff to be available. RFSB lets me make things available without introducing overhead. 

I add myself as a user.

```
addLCM := &rfsb.UserResource{
    User:  "lcm",
    UID:   1000,
    GID:   1000,
    Home:  "/home/lcm",
    Shell: "/bin/bash",
}
rfsb.Register("addLCM", addLCM)
```

I add my key.

```
addLCMSSHKey := &rfsb.FileResource{
  Path:     "/home/lcm/.ssh/authorized_keys",
  Mode:     0400,
  Contents: `ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAL5YH0a+pKd8E8Be97+gN/kn+U71JCapIH8uysrecKB lcm@lcm-mbp`,
  UID:      addLCM.UID,
  GID:      addLCMGroup.GID,
}
rg.When(addLCM).Do("addLCM", addLCMSSHKey)
```

Now I can **log** onto my machines.

### L - Still loving it.

Sometimes I **lose** my machines. In these cases I have to get new machines. When I do, I want to install stuff without complications.

I make the machine mine.

```
$ GOOS=linux go build  -o dist/bootstrap ./example/bootstrap
$ scp dist/bootstrap lcm@schutzenberger.generictestdomain.net:~/bootstrap
$ ssh lcm@schutzenberger.generictestdomain.net ~/bootstrap
```

The bond between me and my machine is now strong.

### L - Loving it more than ever

- But does this scale?
  - You don't scale
  - Does anything scale?
  - Fuck Puppet
  - Bittorrent

- I want features
  - Everything is a file

## Guiding Principles

RFSB is designed to be simple and self sufficent. RFSB binaries should be able to bootstrap systems where /usr/bin has gone walkabouts.

RFSB gets out of your way. By presenting a very thin abstraction, RFSB hopes you will immediately see the weaknesses in its core abstractions, and stop using it.

All features not contained in the current version of RFSB are misfeatures until I decided I need them.

RFSB has a DAG and feels guilty about it.

# Cool.

Check out `example/bootstrap/main.go` for an example of 100% of the functionality that rfsb supports..