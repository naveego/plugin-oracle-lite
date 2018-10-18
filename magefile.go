// +build mage

package main

import (
	"github.com/naveego/ci/go/build"
	"github.com/naveego/dataflow-contracts/plugins"
	"github.com/naveego/plugin-oracle/version"
	"os"
	"runtime"
)

func Build() error {
	cfg := build.PluginConfig{
		Package: build.Package{
			VersionString: version.Version.String(),
			PackagePath:   "github.com/naveego/plugin-oracle",
			Name:          "plugin-oracle",
			Shrink:        true,
			CGOEnabled: true,
		},
		Targets: []build.PackageTarget{
			{
				Arch: runtime.GOARCH,
				OS:   runtime.GOOS,
			},
		},
	}

	err := build.BuildPlugin(cfg)
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
