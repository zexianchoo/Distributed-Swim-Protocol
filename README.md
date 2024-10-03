# SWIM Membership Protocol Distributed System
Distributed server connected with a membership list. Built on the SWIM protocol, with ping-ack as well as ping-ack + suspicion.
Built entirely in golang, and grpc.


# Setup

1. The introducer's hostname is hardcoded in server/constants.go. Change this to the host which will be dedicated as the introducer.

``` INTRODUCER_HOST = "<insert_introducer_ip_here>" ```

2. After changing this constant, cd to /server and start up the introducer with 

```
go run . -i
```

3. For non-introducers, cd to /server and run 
``` 
go run .
```

There is a tmux script provided for ease of running multiple servers. 

```
bash tmux.sh
```
# Stdin commands:
----

There are `stdin` commands that you can run after the server is connected and started up:

``` go

list_mem: prints other hostnames in sorted order, as well as their states.

enable_sus: enable suspicion on all servers. This only needs to be called on one server, and will propagate to the rest.

disable_sus: disables suspicion on all servers. This only needs to be called on one server, and will propagate to the rest.

status_sus: prints suspicion status of this server

list_self: prints current hostname

leave: Leaves the network

exit: calls exit()
```

