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

## tokengen gen fatoken

```
Usage:
  tokengen gen fabtoken [flags]

Flags:
  -a, --auditors strings   list of auditor keys in the form of <MSP-Dir>:<MSP-ID>
      --cc                 generate chaincode package
  -h, --help               help for fabtoken
  -s, --issuers strings    list of issuer keys in the form of <MSP-Dir>:<MSP-ID>
  -o, --output string      output folder (default ".")

```

### dlog public parameters

```
Usage:
  tokengen gen dlog [flags]

Flags:
  -a, --auditors strings   list of auditor keys in the form of <MSP-Dir>:<MSP-ID>
  -b, --base int           base is used to define the maximum quantity a token can contain as Base^Exponent (default 100)
      --cc                 generate chaincode package
  -e, --exponent int       exponent is used to define the maximum quantity a token can contain as Base^Exponent (default 2)
  -h, --help               help for dlog
  -i, --idemix string      idemix msp dir
  -s, --issuers strings    list of issuer keys in the form of <MSP-Dir>:<MSP-ID>
  -o, --output string      output folder (default ".")
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