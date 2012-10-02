tfe
====

A simple http load balancer that forwards requests to downstream based on certain rules.

Currently in this project there's some generic load balancing code, should move out. The interface
used is Serve(interface{})(interface{}, error), similar to Finagle. It seems make sense to unifying
with how Go's doing rpc.
