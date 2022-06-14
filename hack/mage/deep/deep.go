package deep

import (
	"fmt"

	"github.com/magefile/mage/mg"
)

type Deep mg.Namespace

// Deep namespace
func (Deep) Namespace() {
	fmt.Println("namespace in namespace")
}
