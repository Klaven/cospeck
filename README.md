# cospeck
a speed test for contaienr runtimes COntainer SPeed chECK

This is a pre-alpha bit of code. use it at your own risk.

Huge shoutout too the work done https://github.com/estesp/bucketbench while this project does not seem maintained anylonger, it has provided me the base of the cri runtime implementation. Thank you. I would not consider this a "fork" as it is quite different

status: 

Only working command is:
cospeck test general

Currently the output is hard to read, sorry.

How to install?
clone the repo wherever you want.
`make install`


Goals:

As a general testing framework for containers here are some things I would like to include in it:

requirements:
 1) All tests should be able to be run with a custom {pod|container} as long as it is able to be pulled. but they should run with a default setup if not provided with one

 1) Ability to see the performance when {creating|running|shutting down} a {pod|container}
 - memory
 - cpu
 - time
 - network?
 - file?
 - pids?

 1) Ability to perform "nodebuster" 
 this is the ability to create a bunch of `cri` objects to test how many can be made on a node. should be able to use a custom pod def so people can see how many of that pods can run well on a kubernetes node

 1) Ability to test image pull times


### Streatch Goals

Eventually it would be nice if you had multiple container runtimes installed and configured if it could switch between them for you. to let you see if there was a differance between 


### Examples

Docker:
 
sudo ./out/cospeck test general --pod-configfile=./config/pod.yaml --runtime=/var/run/containerd/containerd.sock --cgroup-path=/system.slice/docker.service

Crio:
sudo ./out/cospect test general --pod-configfile=./config/pod.yaml
