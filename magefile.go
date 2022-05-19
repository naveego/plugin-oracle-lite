// +build mage

package main

import (
	"fmt"
	"github.com/magefile/mage/sh"
	"github.com/naveego/ci/go/build"
	"github.com/naveego/dataflow-contracts/plugins"
	"github.com/naveego/plugin-oracle/version"
	"os"
)

func Build() error {

	if err := BuildLinux(); err != nil {
		return err
	}

	if err := BuildWindows(); err != nil {
		return err
	}

	return nil
}

func BuildWindows() error {

	os.Setenv("CC", "x86_64-w64-mingw32-gcc")
	os.Setenv("CXX", "x86_64-w64-mingw32-g++")

	cfg := build.PluginConfig{
		Package: build.Package{
			VersionString: version.Version.String(),
			PackagePath:   "github.com/naveego/plugin-oracle",
			Name:          "plugin-oracle-lite",
			Shrink:        true,
			CGOEnabled: true,
			BuildArgs: []string{"--ldflags", "-w -s"},
		},
		Targets: []build.PackageTarget{
			build.TargetWindowsAmd64,
		},
	}

	err := build.BuildPlugin(cfg)

	if err != nil {
		return err
	}

	os.Unsetenv("CC")
	os.Unsetenv("CXX")

	outZip := fmt.Sprintf("build/outputs/windows/amd64/plugin-oracle-lite/%s/package.zip", version.Version.String())
	finalZip := fmt.Sprintf("build/outputs/plugin-oracle-lite_%s_windows_amd64.zip", version.Version.String())

	return sh.Copy(finalZip, outZip)

	return err
}

func BuildLinux() error {
	cfg := build.PluginConfig{
		Package: build.Package{
			VersionString: version.Version.String(),
			PackagePath:   "github.com/naveego/plugin-oracle",
			Name:          "plugin-oracle-lite",
			Shrink:        true,
			CGOEnabled: true,
			BuildArgs: []string{"--ldflags", "-w -s"},
		},
		Targets: []build.PackageTarget{
			build.TargetLinuxAmd64,
		},
	}

	err := build.BuildPlugin(cfg)
	if err != nil {
		return err
	}

	outZip := fmt.Sprintf("build/outputs/linux/amd64/plugin-oracle-lite/%s/package.zip", version.Version.String())
	finalZip := fmt.Sprintf("build/outputs/plugin-oracle-lite_%s_linux_amd64.zip", version.Version.String())

	return sh.Copy(finalZip, outZip)

	return err
}

// func Build() error {
// 	for _, target := range []build.PackageTarget{
// 		build.TargetLinuxAmd64,
// 		build.TargetDarwinAmd64,
// 		build.TargetWindowsAmd64,
// 	} {
// 		oci8Path := fmt.Sprintf("./build/oracle/contrib/oci8_%s_%s.pc", target.OS, target.Arch)
//
// 		err := sh.Copy("/usr/share/pkgconfig/oci8.pc", oci8Path)
// 		if err != nil {
// 			return err
// 		}
//
// 		err = buildTarget(target)
// 		if err != nil {
// 			return err
// 		}
// 	}
//
// 	return nil
// }
//
// func buildTarget(target build.PackageTarget) error {
// 	cfg := build.PluginConfig{
// 		Package: build.Package{
// 			VersionString: version.Version.String(),
// 			PackagePath:   "github.com/naveego/plugin-oracle",
// 			Name:          "plugin-oracle",
// 			Shrink:        true,
// 		},
// 		Targets: []build.PackageTarget{
// 			target,
// 		},
// 	}
//
// 	err := build.BuildPlugin(cfg)
// 	return err
// }


func PublishBlue() error {
	os.Setenv("UPLOAD", "blue")
	return Build()
}


func GenerateGRPC() error {
	destDir := "./internal/pub"
	return plugins.GeneratePublisher(destDir)
}
