# Nxs-zbxscr

The better way to monitoring multiple instances of same service with Zabbix (e.g. on host system or/and Docker containers) on one node, it is a using the Zabbix autodiscovery mechanism.
In this case at the Zabbix-agent side must be an application implements of following actions:
* `Discovery`: presents to the Zabbix-server information about the service instances at the node (in JSON format).
* `Metric`: after the discovery has been executed and metrics (the 'items' in Zabbix terms) from prototypes has been created for each instance of monitored service, the metrics starts to collect the data. Every metric check it's a request to the Zabbix-agent and call associated application (or script). An application in turn makes a request to an appropriate instance of the service for data obtaining.

If a monitored service represents a lot of metrics, each check will be made a request. It may spawn a redundant load and distort the data. To prevent this you can set up and use a cache to reduce interactions with the service.

This Go package provides the tools to write an applications for Zabbix-agent checks. This applications oriented to work with Zabbix autodiscovery.

The package includes following kinds:
* `Instance`
* `Actions`
* `Exporter`
* `Cache`

## Instance

In common case application config file contains an array of instances for certain monitored service. Each instance block describes a way to interact with service instance for obtain a specified metric data.

## Actions

To use this package you need to implement a functions for following actions:
* `Discovery`: function implements an instances discovery. This function must return the slice of elements to be sent to Zabbix server to automatically create items, triggers, and graphs for different entities.
* `CheckConf`: function implements a check config for syntax errors, instances duplicates, etc.
* `CheckAlive`: implements a check for an ability to obtain the data from specified instance
* `Metric`: implements a data obtaining for specified metric

## Exporter

All interactions with monitored service aimed to obtain the data must be made through `Exporter`. If you are using a cache, this fucntion will be automatically called when instance cache does not exist or it's outdated.

## Cache

If the service represents a lot of metrics, you can reduce number of interactions them by using cache. If this feature is used, first request for any metric generate a cache and all subsequent requests will be obtained from them. Further, first metric request after cache is outdated will be update them, and so on.

## How to use

To develop the application using this package you must implement following functions:
* `Discovery`
* `Metric`
* `Exporter`
* `CheckConf`
* `CheckAlive`

See the tests for examples.
