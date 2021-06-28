package events

import v1 "k8s.io/api/apps/v1"

type StatefulSetEvent func(sts *v1.StatefulSet)
