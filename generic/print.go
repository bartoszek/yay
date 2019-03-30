package generic

import (
	"fmt"

	"github.com/Jguer/yay/v9/conf"
)

// Arrow used in display
const Arrow = "==>"

// SmallArrow using in prints
const SmallArrow = " ->"

const (
	RedCode     = "\x1b[31m"
	GreenCode   = "\x1b[32m"
	YellowCode  = "\x1b[33m"
	BlueCode    = "\x1b[34m"
	MagentaCode = "\x1b[35m"
	CyanCode    = "\x1b[36m"
	BoldCode    = "\x1b[1m"

	ResetCode = "\x1b[0m"
)

// Human method returns results in human readable format.
func Human(size int64) string {
	floatsize := float32(size)
	units := [...]string{"", "Ki", "Mi", "Gi", "Ti", "Pi", "Ei", "Zi", "Yi"}
	for _, unit := range units {
		if floatsize < 1024 {
			return fmt.Sprintf("%.1f %sB", floatsize, unit)
		}
		floatsize /= 1024
	}
	return fmt.Sprintf("%d%s", size, "B")
}

func stylize(startCode, in string) string {
	if conf.UseColor {
		return startCode + in + ResetCode
	}

	return in
}

// Red prints red text
func Red(in string) string {
	return stylize(RedCode, in)
}

// Green prints green text
func Green(in string) string {
	return stylize(GreenCode, in)
}

// Yellow prints Yellow text
func Yellow(in string) string {
	return stylize(YellowCode, in)
}

// Blue prints blue text
func Blue(in string) string {
	return stylize(BlueCode, in)
}

// Cyan prints cyan text
func Cyan(in string) string {
	return stylize(CyanCode, in)
}

// Magenta prints magenta text
func Magenta(in string) string {
	return stylize(MagentaCode, in)
}

// Bold boldens the text
func Bold(in string) string {
	return stylize(BoldCode, in)
}

// ColourHash colours text using a hashing algorithm. The same text will always produce the
// same colour while different text will produce a different colour.
func ColourHash(name string) (output string) {
	if !conf.UseColor {
		return name
	}
	var hash uint = 5381
	for i := 0; i < len(name); i++ {
		hash = uint(name[i]) + ((hash << 5) + (hash))
	}
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", hash%6+31, name)
}
