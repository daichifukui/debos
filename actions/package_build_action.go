/*
Package Build

Build package with patch

Yaml syntax:
 - action: pkg_build
   package: package name
   version: main | backports | contrib | v1.34
   codename: jessie | stretch | buster
   architecture: i386 | x86_64 | armhf | armel | arm64
   patch: patch files
   destination: directory
   install: true | false
   sign_key: signed key
   
Mandatory properties:
- package -- name of the package to be built

- version -- version of the package

- codename -- codename of the target Debian

Optional properties:
- architecture

- patch

- destination

- install

- sign_key

*/
package actions

import (
	"log"
	"syscall"
	"fmt"

	"github.com/go-debos/debos"
)

type PackageBuildAction struct {
	debos.BaseAction `yaml:",inline"`
	Package		 string
	version		 string
	codename	 string
	architecture	 string
	patch		 string
	installation	 string
	install		 bool
	sign_key	 string
}

func (pb *PackageBuildAction) Run(context *debos.DebosContext) error {
	pb.LogStart()
	return debos.Command{}.Run("Building", "dpkg-buildpackage", "-uc -us")
}
