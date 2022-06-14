package dependencies

import (
	"fmt"
	"os"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

func Aaa() error {
	fmt.Println("aaa")
	time.Sleep(1 * time.Second)
	return fmt.Errorf("aaa")
}

func Bbb() error {
	fmt.Println("bbb")
	return nil
}

// Ccc stderr redirected to stdout, no exit code
func Ccc() error {
	sh.Exec(nil, os.Stdout, os.Stdout, "hack/build/error.sh")
	return nil
}

// Ddd stderr redirected to stdout
func Ddd() error {
	ran, err := sh.Exec(nil, os.Stdout, os.Stdout, "hack/build/error.sh")
	if !ran {
		return fmt.Errorf("error.sh not found/not executable")
	}
	return err
}

// Eee stderr not redirected
func Eee() {
	sh.Run("hack/build/error.sh")

}

// Fff depends on eee
func Fff() {
	mg.Deps(Eee)
	fmt.Println("fff")
}

// Ggg depends on Aaa (fail) and Bbb (pass)
func Ggg() {
	mg.Deps(Aaa, Bbb)
	fmt.Println("ggg")
}

// Hhh serial dependence on Aaa (fail) and Bbb (pass)
func Hhh() {
	mg.SerialDeps(Aaa, Bbb)
	fmt.Println("ggg")
}
