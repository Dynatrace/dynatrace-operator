package deep

import (
	"fmt"

	"github.com/magefile/mage/mg"
)

type Deep mg.Namespace

// Namespace a simple target to present namespaces
func (Deep) Namespace() {
	fmt.Println("namespace in namespace")
}
