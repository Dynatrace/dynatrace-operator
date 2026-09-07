// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dtimage "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	imagemock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/image"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_Reconciler_Reconcile(t *testing.T) {
	edgeConnect := createBasicEdgeConnect(t)
	fakeImageClient := imagemock.NewClient(t)
	fakeImageClient.EXPECT().GetComponentLatestInfo(anyCtx, dtimage.EdgeConnect, mock.Anything).
		Return(&dtimage.Info{URI: fakeImageURI}, nil)

	reconciler := NewReconciler(fake.NewClient(), staticImageClientProvider(fakeImageClient), timeprovider.New(), edgeConnect)

	require.NotNil(t, reconciler)
	require.NoError(t, reconciler.Reconcile(t.Context()))
}
