package cri

import (
	"io"
	"io/ioutil"
	"strings"
	"testing"
)

func TestParseYamlFile(t *testing.T) {

	bytes, err := ioutil.ReadAll(pod)
	pod, containers, err := ParseYamlFileWithPodConfig(bytes, sandboxConfig, containerConfig)
	if err != nil {
		t.Errorf("Error parsing YAML file: %s\n", err)
	}

	if len(containers) != 1 {
		t.Errorf("Expected exactly one container found %d", len(containers))
	}

	if containers[0].Image.Image != "docker.io/library/alpine:latest" {
		t.Errorf("Expected image 'docker.io/library/alpine:latest' but found '%s'", containers[0].Image.Image)
	}

	if pod.Metadata.Name != "basic-pod" {
		t.Errorf("Expected Pod Name 'basic-pod' found '%s'", pod.Metadata.Name)
	}

}

var pod io.Reader = strings.NewReader(`apiVersion: v1
kind: Pod
metadata:
  name: basic-pod
spec:
  containers:
    - name: web
      image: docker.io/library/alpine:latest
      command:
        - sleep
        - "5000"
      ports:
        - name: web
          containerPort: 80
          protocol: TCP`)

var containerConfig io.Reader = strings.NewReader(`{
	"metadata": {
		"name": "cospeck",
		"attempt": 1
	},
	"image": {
		"image": "docker.io/library/alpine:latest"
	},
	"command": [
		"/bin/ls"
	],
	"args": [],
	"working_dir": "/",
	"envs": [
		{
			"key": "PATH",
			"value": "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
		},
		{
			"key": "TERM",
			"value": "xterm"
		}
	],
	"privileged": true,
	"log_path": "",
	"stdin": false,
	"stdin_once": false,
	"tty": false,
	"linux": {
		"resources": {
			"cpu_period": 10000,
			"cpu_quota": 20000,
			"cpu_shares": 512,
			"oom_score_adj": 30
		},
		"security_context": {
			"readonly_rootfs": false,
			"selinux_options": {
			    "user": "system_u",
			    "role": "system_r",
			    "type": "svirt_lxc_net_t",
			    "level": "s0:c4,c5"
			},
			"capabilities": {
				"add_capabilities": [
					"setuid",
					"setgid"
				],
				"drop_capabilities": [
				]
			}
		}
	}
}`)

var sandboxConfig io.Reader = strings.NewReader(`{
	"metadata": {
		"name": "cospeck",
		"uid": "cospeck-test-cri",
		"namespace": "cospeck.test.cri",
		"attempt": 1
	},
	"hostname": "crioctl_host",
	"log_directory": "",
	"dns_config": {
		"searches": [
			"8.8.8.8"
		]
	},
	"port_mappings": [],
	"resources": {
		"cpu": {
			"limits": 3,
			"requests": 2
		},
		"memory": {
			"limits": 50000000,
			"requests": 2000000
		}
	},
	"labels": {
		"group": "test"
	},
	"annotations": {
		"owner": "hmeng",
		"security.alpha.kubernetes.io/sysctls": "kernel.shm_rmid_forced=1,net.ipv4.ip_local_port_range=1024 65000",
		"security.alpha.kubernetes.io/unsafe-sysctls": "kernel.msgmax=8192" ,
		"security.alpha.kubernetes.io/seccomp/pod": "unconfined"
	},
	"linux": {
		"security_context": {
			"namespace_options": {
				"host_network": false,
				"host_pid": false,
				"host_ipc": false
			},
			"selinux_options": {
				"user": "system_u",
				"role": "system_r",
				"type": "svirt_lxc_net_t",
				"level": "s0:c4,c5"
			}
		}
	}
}`)
