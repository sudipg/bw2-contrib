# Drivers

A guide on how to construct BW2 drivers.

Drivers abstract away the details of communicating to underlying hardware and
services to present a uniform interface.

This document should be consumed after reading up on [Bosswave services](https://github.com/immesys/bw2/wiki/Services).

## URIs

Drivers and services in BOSSWAVE publish and subscribe to URIs.

In this section we establish a set of idioms for URI construction for these drivers.
We can divide URIs into several components:

```
<baseuri>/<service>/<instance>/<interface>/{signal,slot}/<name>
```

* **`baseuri`**: the base URI, starting with a namespace (e.g. `410.dev`) and
  probably some additional subdivision, e.g. `410.dev/sodahall/drivers`. This
  is the URI information provided to the driver during its initial
  configuration because this describes where in the administrative hierarchy
  the driver is deployed.  The driver will generate all URIs from this point
  starting with the `service` component below.

* **`service`**: the name of the service, using the `s.` notation, so a "LIFX
  driver" might be named `s.lifx`. The service name will be dictated by the
  driver.

  * **TODO**: we want a list of all available service names and links to the drivers

* **`instance`**:  combined with `interface` below, describes a unique instance
  of an interface. This is not the appropriate place to attach descriptive
  metadata, so the `instance` name should just be enough to distinguish between
  instances.  For example, a plug strip service (e.g. `s.plugstrip`) might have
  multiple plugs that each present a binary actuator interface (e.g.
  `i.binary_actuator`). In this case, we might name the instances by their plug
  number: `0`, `1`, etc.

* **`interface`**: the name of the interface uses the `i.` notation, so a
  hue-saturation-brightness interface for a light could be named `i.hsb-light`.
  An interface defines a set of `signal`s and `slot`s for publishing and
  consuming data respectively.

  * **TODO**: define interfaces somewhere

* **signal**s and **slot**s: URI endpoints are either a **signal** (output) or a **slot** (input).
  A signal emits new readings and state from the service. Processes subscribe to a signal.
  Signals can be used for advertising state changes, new sensor readings, etc.

  A slot is a URI to which the service subscribes and is how other processes can send
  data to a service.

* **`name`**: this is the name of the signal or slot on the interface. The
  combination of `signal`/`slot` and `name` is defined by the interface.

### Example: Creating a Service and Interfaces

Here is some example code for how we might start defining a driver service for Weather Underground

```go
package main

// retrieve the BOSSWAVE bindings
import (
	bw2 "gopkg.in/immesys/bw2bind.v5"
    "os"
    "time"
)

func main() {
    // connects to the default BOSSWAVE router running locally
    client := bw2.ConnectOrExit("")

    // intelligent defaults for publishing/subscribing
    client.OverrideAutoChainTo(true)

    // need an entity file to define "who" this driver is
    // here we grab the file name from the environment, but we could also
    // hard code it or fetch it from a configuration file
    client.SetEntityFromEnvironOrExit(os.Getenv("BW2_DEFAULT_ENTITY"))

    // the root of the URI for the driver, i.e. "where" it is deployed
    baseuri := "scratch.ns/mydrivers"

    // creates the URI "scratch.ns/mydrivers/s.weatherunderground")
    service := client.RegisterService(baseuri, "s.weatherunderground")
    // creates the URI "scratch.ns/mydrivers/s.weatherunderground/Berkeley/i.weather"
    // "Berkeley" is the instance, "i.weather" is the interface
    iface := service.RegisterInterface("Berkeley", "i.weather")

    // attach metadata here (see below)
}
```

## Metadata

Metadata is descriptive data attached to a service, an instance, an interface, or a signal/slot.
Services will persist data at special BOSSWAVE URIs.

All resources in BOSSWAVE are identified with some URI that acts as a kind of
"pointer": this is true both for topics acting as data transport channels
between publishers and subscribers (as with the signals and slots above) as
well as *persisted data*.

Any message published on a URI will be delivered to all subscribers, but in
some cases a process may want to see whatever message was last published on a
URI without having to wait for something to publish on that URI. Persisted
messages "live" on a URI and can be retrieved using a [BOSSWAVE query](https://godoc.org/gopkg.in/immesys/bw2bind.v5#BW2Client.Query)
(not a subscription).

By convention, metadata attached to a URI `<u>` should be persisted at
`<u>/!meta/<keyname>`. Because the persisted objects are full messages, they
can contain lists of Payload Objects of various types.

### Schemas

The string `!meta` in a URI identifies that all parts of the URI following are
part of the "meta" schema. In the "meta schema", keys are the portions of the URI
follwing `!meta`, and values are a "Simple Metadata" ([PONum 2.0.3.1/32](https://github.com/immesys/bw2_pid/blob/master/allocations.yaml#L201-L207))
struct consisting of a *string* value and a timestamp of when that metadata
was set.

* **TODO**: define the semantics for a set of metadata schemas

### Metadata Placement in URIs

In a service URI, we expect to place metadata at the following locations (`^`)
```
<baseuri>/<service>/<instance>/<interface>/{signal,slot}/<name>
                   ^          ^           ^                    ^
               service MD  instance MD  iface MD             signal/slot MD
```

Ideally, we'd place metadata in the URI where it is most helpful.

* `<basuri>/<service>/!meta/<key>`: metadata describing the service itself,
  such as whether or not it has to do with lighting or where it is deployed.

* `<baseuri>/<service>/<instance>/!meta/<key>`: metadata describing an instance
  of a service, but more generic than the capabilities of that instance (which would
  be under the purview of interface-metadata). For the "Berkeley" instance of the WeatherUnderground
  service described above, possible metadata might include the state, country and zipcode of the
  actual city, to further distinguish *which* Berkeley we mean. For a lighting instance, this may
  describe the physical owner of the device.

* `<baseuri>/<service>/<instance>/<interface>/!meta/<key>`: metadata describing
  the specifics/semantics of an interface.

* `<baseuri>/<service>/<instance>/<interface>/{signal/slot}/<name>/!meta/<key>`: metadata attached
  to a particular signal or slot. The primary use case here would be the use of attaching `!meta/archive`
  (with a value of a URI) indicating to an archival service that all messages sent on a signal or received
  on a slot should be archived (assuming appropriate permissions, of course).

  **note**: placing metadata here is not typical. Most metadata should be at the service, instance,
  or interface level.

**note**: in principle, these should all work with other metadata schemas other than `!meta`

### Example

Now, lets add some metadata to our Weather Underground driver. This code demonstrates
*how* metadata can be attached to a URI, and does not aim to inform *what* form this metadata
should take.

```go
    // attach metadata to the service to describe it as a web service

    // using service.SetMetadata handles packing the metadata object for us

    // creates the URI "scratch.ns/mydrivers/s.weatherunderground/!meta/type"
    service.SetMetadata("type", "Web Service")
    // creates the URI "scratch.ns/mydrivers/s.weatherunderground/!meta/url"
    service.SetMetadata("url", "https://www.wunderground.com/weather/api/")
    // there is also iface.SetMetadata

    // attach information to the instance "Berkeley" -- here we cannot use the library
    // functions because none exist for this, but we can emulate its behavior easily enough.

    // The following code is what happens "under the covers" when we call service.SetMetadata
    // or iface.SetMetadata

    // create a payload object to be persisted describing which State our instance is in
    po := bw2.CreateMetadataPayloadObject(&bw2.MetadataTuple{
        Value:  "California",             // the metadata value
        Timestamp: time.Now().UnixNano(), // the time at which the metadata is set
    })
    // create "scratch.ns/mydrivers/s.weatherunderground/Berkeley/!meta/state"
    state_uri := service.FullURI() + "Berkeley/!meta/state"
    client.PublishOrExit(&bw2.PublishParams{
        PayloadObjects: []bw2.PayloadObject{po}, // attach our metadata to the message
        URI:            state_uri,               // where the message will be sent
        Persist:        true,                    // make sure to persist the message
    })

```

**TODO**: clarify: are hierarchical metadata keys ok, or does this break the permissions model?

## Permissions

For our deployed driver to be useful, permissions must exist for:
* the driver to publish data on a set of URIs (signals)
* the driver to subscribe to data on a set of URIs (slots)
* other processes to subscrbe to data published by the driver (signals)
* other processes to publish to the driver (slots)

**Driver Permissions**: the easiest method is to grant the driver full permissions on the
URI defined by the deployment namespace and the service name:

* permissions: `LPC*`
* URI: `<namespace>/<service>/*`
* TTL: typically 0 -- the driver does not need to grant permissions to anyone

For example:

```
bw2 mkdot --from <granting entity> --to <driver entity> --uri scratch.ns/mydrivers/s.weatherunderground/*
```

**Usage Permissions**: there is more flexibility here because of how we may
want to divide up permissions. The URI construction of services allows us this flexibility.

We can (separately or together):
* grant full subscribe access to signals: We use `+` to fuzz the instance and interface
    * permissions: `LC` (L is optional)
    * URI: `<namespace>/<service>/+/+/signal/*`
* grant full publish access to slots:
    * permissions: `P`
    * URI: `<namespace>/<service>/+/+/slot/+`
* grant subscription access to all interfaces of a certain type:
    * permissions: `C`
    * URI: `*/i.interfacename/signal/+`

**TODO**: verify that these are the correct patterns

## Deployment

**TODO**: spawnpoint permissions

## Archiving