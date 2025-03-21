

image=${2}
annotation=${2}

podman pull $image

podman manifest inspect $image


for
  podman manifest annotate --annotation $image $digest


