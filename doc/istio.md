# How we handle the istio integration

For the `operator` and it's deployed components to work in an _istio environment_ in `REGISTRY_ONLY` mode it naturally needs a few `ServiceEntries` and `VirtualServices`.

We can differentiate between 2 "sources" of endpoints/hosts we have to worry about:

1. The `APIURL` defined in the `DynaKube`.
    - The operator uses to communicate with the Dynatrace Environment.
    - Should be handled before anything, otherwise the use would have to configure istio for this url.
2. The `Communication Hosts` of the `OneAgent`.
    - We get it from Dynatrace Environment periodically. (using the `APIURL`)
    - May dynamically change overtime, **main reason** for the integration. (otherwise there could be a "setup once" solution, no fancy operator required)

We can also differentiate between 2 "types" of endpoints/hosts, which is important because you can't mix them in a `ServiceEntry`, you have to create 1 for each type: (A `ServiceEntry` can have multiple hosts listed in it, but has to be the same "type")

1. IP based
    - Needs **no** corresponding `VirtualService`
2. FQDN(Fully qualified domain name) based
    - Needs corresponding `VirtualService`

```mermaid
---
title: Simplified istio reconcile flow
---
flowchart LR
    dynakube[Dynakube\nName: test\nEnableIstio: true]
    subgraph operator
        direction TB
        reconcile-istio-for-api-url
        reconcile-connectionInfo...
        reconcile-istio-communication-hosts
        reconcile-everything-else...
    end
    subgraph for-api-url
        direction TB
        se1{{ServiceEntry\nName: test-dk-fqdn-operator \nPurpose: For the api-url}}
        vs1{{VirtualService\nName: test-dk-fqdn-operator \nPurpose: For the api-url}}
        se1 -.- vs1
    end
    subgraph for-communication-hosts
        direction TB
        se2{{ServiceEntry\nName: test-dk-ip-oneagent \nPurpose: For oneagent IP-based communication-hosts}}
        se3{{ServiceEntry\nName: test-dk-fqdn-oneagent \nPurpose: For oneagent FQDN-based communication-hosts}}
        vs2{{VirtualService\nName: test-dk-fqdn-oneagent \nPurpose: For oneagent FQDN-based communication-hosts}}
        se3 -.- vs2
        se2 ~~~ se3
    end


    exp1([User doesn't need to configure istio for the API Url by hand]):::good-green
    exp2([Update logic is simple as the names are static + names mean something\nincase of error, we actually what we are doing]):::good-green

    dynakube-->operator
    reconcile-istio-for-api-url-->for-api-url
    reconcile-istio-communication-hosts-->for-communication-hosts
    for-api-url --> exp1
    for-api-url --> exp2
    for-communication-hosts --> exp2

    classDef good-green stroke:#0f0
```
