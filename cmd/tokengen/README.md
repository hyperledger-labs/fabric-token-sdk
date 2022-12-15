# Tokengen

`tokengen` is an utility for generating Fabric Token-SDK material. 
It is provided as a means of preconfiguring public parameters, token chaincode, and so. 
It would normally not be used in the operation of a production network.

## Syntax

The `tokengen` command has five subcommands, as follows:

- artifacts
- certifier-keygen
- gen
- help
- version

## tokengen artifacts

This command is used to centrally generate key material and configuration files.
It takes in input a `topology` file, in `yaml` format, that describes the topologies of the networks involved.
An example can be found [`here`](./samples/topology/fungible.yaml). 
Topology files can be edited directly or they can be generated programmatically as shown [`here`](./samples/topology/fungible.go). 

```
Read topology from file and generates artifacts.

Usage:
tokengen artifacts [flags]

Flags:
-h, --help              help for artifacts
-o, --output string     output folder (default "./testdata")
-p, --port int          host starting port (default 20000)
-t, --topology string   topology file in yaml format
```

## tokengen certifier-keygen

```
Gen Token Certifier Key Pair.

Usage:
  tokengen certifier-keygen [flags]

Flags:
  -d, --driver string   driver (dlog) (default "dlog")
  -h, --help            help for certifier-keygen
  -o, --output string   output folder (default ".")
  -p, --pppath string   path to the public parameters file
```

## tokengen gen

The `tokengen gen` command has two subcommands, as follows:

- fatoken: generates the public parameters for the fabtoken driver
- dlog: generates the public parameters for the dlog driver

## tokengen gen fabtoken

```
Usage:
  tokengen gen fabtoken [flags]

Flags:
  -a, --auditors strings   list of auditor MSP directories containing the corresponding auditor certificate
      --cc                 generate chaincode package
  -h, --help               help for fabtoken
  -s, --issuers strings    list of issuer MSP directories containing the corresponding issuer certificate
  -o, --output string      output folder (default ".")

```

The public parameters are stored in the output folder with name `fabtoken_pp.json`.

### tokengen gen dlog

```
Usage:
  tokengen gen dlog [flags]

Flags:
  -a, --auditors strings   list of auditor MSP directories containing the corresponding auditor certificate
  -b, --base int           base is used to define the maximum quantity a token can contain as Base^Exponent (default 100)
      --cc                 generate chaincode package
  -e, --exponent int       exponent is used to define the maximum quantity a token can contain as Base^Exponent (default 2)
  -h, --help               help for dlog
  -i, --idemix string      idemix msp dir
  -s, --issuers strings    list of issuer MSP directories containing the corresponding issuer certificate
  -o, --output string      output folder (default ".")
``` 

The public parameters are stored in the output folder with name `zkatdlog_pp.json`.

## tokengen pp

The `tokengen pp` command has the following subcommands:

- print: Inspect public parameters

### tokengen pp print

```
Usage:
  tokengen pp print [flags]

Flags:
  -h, --help           help for print
  -i, --input string   public param file
```

## tokengen help

```
Help provides help for any command in the application.
Simply type tokengen help [path to command] for full details.

Usage:
  tokengen help [command] [flags]

Flags:
  -h, --help   help for help
```

## tokengen version

```
Print current version of tokengen.

Usage:
  tokengen version [flags]

Flags:
  -h, --help   help for version
```