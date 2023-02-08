package v1beta1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsRequestOutdated(t *testing.T) {
	t.Run(`returns true if the request is outdated`, func(t *testing.T) {
		outdated := metav1.NewTime(time.Now().Add(-time.Minute * 16))
		assert.True(t, IsRequestOutdated(outdated))
	})
	t.Run(`returns false if the request is not outdated`, func(t *testing.T) {
		notOutdated := metav1.Now()
		assert.False(t, IsRequestOutdated(notOutdated))
	})
}
