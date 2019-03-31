package dep

import (
	"github.com/mikkeloscar/aur"
)

// Base represents a RPC package slice
type Base []*aur.Pkg

// Pkgbase returns the PackageBase of the first package
func (b Base) Pkgbase() string {
	return b[0].PackageBase
}

// Version returns the Version of the first package
func (b Base) Version() string {
	return b[0].Version
}

// URLPath returns the URLPath of the first package
func (b Base) URLPath() string {
	return b[0].URLPath
}

// Pretty print a set of packages from the same package base.
// Packages foo and bar from a pkgbase named base would print like so:
// base (foo bar)
func (b Base) String() string {
	pkg := b[0]
	str := pkg.PackageBase
	if len(b) > 1 || pkg.PackageBase != pkg.Name {
		str2 := " ("
		for _, split := range b {
			str2 += split.Name + " "
		}
		str2 = str2[:len(str2)-1] + ")"

		str += str2
	}

	return str
}
