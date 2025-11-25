# Standard Docker Compose Casting

This directory contains the **standard casting** for running SigNoz using **Docker Compose**.  
A casting is a fully assembled deployment bundle generated from reusable configuration molds.


## 1. Moldings (Configuration Templates)

The template configuration files used by this casting are located at:

```bash
foundry/moldings/
```

Each component provides its own set of molds under the **standard** flavour.  
These files contain placeholder variables (e.g. `${VAR}`) which must be filled before deployment.


## 2. Environment Variables

This directory must contain a `.env` file:

```bash
./.env
````

This file provides the variables required to pour the molds.

A minimal example:

```env
ZOOKEEPER_HOST=zookeeper-1
ZOOKEEPER_PORT=2181
CLICKHOUSE_HOST=clickhouse
CLICKHOUSE_PORT=9000
SIGNOZ_HOST=signoz
````

Adjust values as needed for your environment.

## 3. Forge: Generate the Pours

Run the forge to render the molds:

```
./foundry/forge.sh
```

This will:

* Load variables from the `.env` file in this directory
* Render all standard molds
* Write the poured configs into:

```
./pours/
```

These pours are what the casting consumes.

## 4. Run the Standard Casting

After pours are generated, start the deployment with:

```
docker compose up -d
```

The compose file in this directory mounts the necessary files from:

```
../pours/
```
