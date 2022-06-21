package dependencies

import (
	"fmt"
	"os"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Aaa an example for serial/parallel dependency
func Aaa() error {
	fmt.Println("aaa")
	time.Sleep(1 * time.Second)
	return fmt.Errorf("aaa")
}

// Bbb an example for serial/parallel dependency
func Bbb() error {
	fmt.Println("bbb")
	return nil
}

// Parallel dependence on Aaa (fail) and Bbb (pass), Bbb is executed.
func Parallel() {
	mg.Deps(Aaa, Bbb)
	fmt.Println("parallel")
}

// Serial dependence on Aaa (fail) and Bbb (pass), Bbb isn't executed.
func Serial() {
	mg.SerialDeps(Aaa, Bbb)
	fmt.Println("serial")
}

// StdoutOnlyNoExitCode stderr redirected to stdout, no exit code
func StdoutOnlyNoExitCode() error {
	sh.Exec(nil, os.Stdout, os.Stdout, "hack/build/error.sh")
	fmt.Println("StdoutOnlyNoExitCode")
	return nil
}

// StdoutOnlyExitCode stderr redirected to stdout, exit code returned
func StdoutOnlyExitCode() error {
	ran, err := sh.Exec(nil, os.Stdout, os.Stdout, "hack/build/error.sh")
	if !ran {
		return fmt.Errorf("error.sh not found/not executable")
	}
	return err
}

// StdStreams stderr not redirected, exit code returned
func StdStreams() error {
	err := sh.Run("hack/build/error.sh")
	fmt.Println("StdStreams")
	return err
}
