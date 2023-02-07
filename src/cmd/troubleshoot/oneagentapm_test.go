package troubleshoot

/*func TestTroubleshootOneAgentAPM(t *testing.T) {
	t.Run("oneagentAPM does not exist in cluster", func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testNamespace,
					UID:  testUID,
				},
			}).
			Build()

		troubleshootCtx := troubleshootContext{context: context.TODO(), apiReader: clt, kubeConfig: rest.Config{}}
		assert.NoError(t, checkOneAgentAPM(&troubleshootCtx))
	})
}*/
