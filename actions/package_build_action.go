/*
Package Build

Build package with patch

Yaml syntax:
 - action: pkg_build
   package: package name
   #version: main | backports | contrib | v1.34 #XXX: not needed, obvious
   #codename: jessie | stretch | buster # XXX: obvious, get from environment variable
   #architecture: i386 | x86_64 | armhf | armel | arm64 #XXX: only native build
   patch: patch files
   #destination: directory #XXX: not needed 
   #install: true | false
   #sign_key: signed key
   
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
	"fmt"
	"syscall"
	"os"
	"path"

	"github.com/go-debos/debos"
)

type PackageBuildAction struct {
	debos.BaseAction `yaml:",inline"`
	Package		 string
	Version		 string
	Codename	 string
	Architecture	 string
	Patch		 string
	Installation	 string
	Install		 bool
	Sign_key	 string
}

func Exists(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}

func (pb *PackageBuildAction) Run(context *debos.DebosContext) error {
	pb.LogStart()
	var options string
	var err error
	var cmd debos.Command

	// XXX: dependent on host
	if ! Exists("/tmp/upper") {
		err = syscall.Mkdir("/tmp/upper",0644)
		if err != nil {
			return fmt.Errorf("Couldn't Mkdir: %v", err)
		}
	}
	// XXX: dependent on host
	if ! Exists("/tmp/work") {
		err = syscall.Mkdir("/tmp/work",0644)
		if err != nil {
			return fmt.Errorf("Couldn't Mkdir: %v", err)
		}
	}

	if err != nil {
		return fmt.Errorf("Coundn't OpenFile:%v", err)
	}
	options = "lowerdir="
	options = options + context.Rootdir
	options = options + ",upperdir=/tmp/upper,workdir=/tmp/work"

	err = syscall.Mount("overlay",context.Rootdir,"overlay",0,options)
	if err != nil {
		return fmt.Errorf("Couldn't Mount: %v", err)
	}

	cmd = debos.NewChrootCommandForContext(*context)
	// HACK
	srclist, err := os.OpenFile(path.Join(context.Rootdir, "etc/apt/sources.list"),
		os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("Coundn't OpenFile:%v", err)
	}
	_, err = srclist.WriteString("deb-src https://deb.debian.org/debian buster main\n")
	if err != nil {
		return fmt.Errorf("Coundn't WriteString:%v", err)
	}
	defer srclist.Close()

	cmd.Run("PkgBuild","apt-get","update")
	cmd.Run("PkgBuild","apt-get","-y","install","dpkg-dev","debhelper")
	cmd.Run("PkgBuild","apt-get","build-dep",pb.Package)
	cmd.AddEnv("DEB_BUILD_OPTIONS=noautodbgsym")
	cmd.Run("PkgBuild","apt-get","source",pb.Package)
	// TODO: apply patch here if required
	var cmdline []string
	if pb.Patch != "" {
		if ! Exists(pb.Patch) {
			return fmt.Errorf("patch file not found: %s",pb.Patch)
		}
		cmdline = []string{"cp "+pb.Patch+" "+context.Rootdir}
		cmdline = append([]string{"sh","-c"}, cmdline...)
		debos.Command{}.Run("PkgBuild", cmdline...)
		cmdline = []string{"patch -p0 < "+"/"+path.Base(pb.Patch)}
		cmdline = append([]string{"sh","-c"}, cmdline...)
		err = cmd.Run("PkgBuild", cmdline...)
		if err != nil {
			return fmt.Errorf("failed to apply patch: %s",pb.Patch)
		}
	}
	cmd.Run("PkgBuild","apt-get","--compile","source",pb.Package)

	fmt.Println("==DEBUG==")
	fmt.Println(cmdline)
	fmt.Println(err)
	fmt.Println("patch file: "+pb.Patch)

	//TODO: copy deb package to artifactdir BEFORE UMOUNT OVERLAYFS
	//TODO: install the deb package AFTER UMOUNT OVERLAYFS
	//TODO: two ways to do that
	//(1) mkdir /tmp/merged and mount overlayfs on it as lowerdir=chroot.dir => then chdir to /tmp/merged
	//(2) mount overlayfs on chroot.dir and move deb package to artifactdir => umount ovalayfs => dpkg -i debpkg
	//(3) mount overlayfs over multiple directories suchas /root, /tmp, /var; build deb package under /root
	//(4) mount ovarlayfs each of /tmp and /var; apt-get --compile under /root; umount overlayfs; dpkg -i; rm deb package under /root 
	//TODO: mv *.dsc *.deb to change name
	defer func() {
		var cmdline []string
		var cmd debos.Command

		cmd = debos.NewChrootCommandForContext(*context)
		cmdline = []string{"cp /tmp/upper/*.deb "+context.Rootdir}
		cmdline = append([]string{"sh","-c"}, cmdline...)
		debos.Command{}.Run("PkgBuild", cmdline...)
		fmt.Println(cmdline)
		cmdline = []string{"dpkg -i *.deb"}
		cmdline = append([]string{"sh","-c"}, cmdline...)
		cmd.Run("PkgBuild", cmdline...)
		cmd.Run("PkgBuild","apt-get","-f","install")
		cmdline = []string{"rm /*.deb "}
		cmdline = append([]string{"sh","-c"}, cmdline...)
		cmd.Run("PkgBuild", cmdline...)
		//cmd.Run("PkgBuild","apt","install","*.deb")
		//cmd.Run("PkgBuild","dpkg","-l","stress")
	} ()

	return err
}

func (pb *PackageBuildAction) Cleanup(context *debos.DebosContext) error {
	var err error
	var cmd debos.Command

	cmd = debos.NewChrootCommandForContext(*context)

	err = syscall.Unmount(context.Rootdir,0)
	if err != nil {
		return fmt.Errorf("Couldn't Unmount: %v", err)
	}

	cmd.Run("PkgBuild","ls","-l")

	err = os.RemoveAll("/tmp/upper")
	if err != nil {
		return fmt.Errorf("Couldn't RemoveAll: %v", err)
	}

	err = os.RemoveAll("/tmp/work")
	if err != nil {
		return fmt.Errorf("Couldn't RemoveAll: %v", err)
	}
	return nil
}
